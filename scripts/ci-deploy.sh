#!/usr/bin/env bash
# CI Pipeline - Deploy Script for FortiGate A/P HA Single Zone
# Sourced by terraform-ci-deploy.sh in the deploy stage
# TERRAFORM_DIR and TF_OUTPUTS_PATH are set by the pipeline before sourcing

set -euo pipefail

echo "============================================"
echo "FortiGate HA Single Zone - Deploy"
echo "============================================"

cd "$TERRAFORM_DIR"
echo "Working directory: $(pwd)"

# Load all TF_VAR_* exports and IC_API_KEY from the shared env loader
# shellcheck source=scripts/ci-env.sh
source "${TERRAFORM_DIR}/scripts/ci-env.sh"

# Validate required variables
REQUIRED_VARS=(
  TF_VAR_IBMCLOUD_API_KEY
  TF_VAR_REGION
  TF_VAR_ZONE
  TF_VAR_VPC
  TF_VAR_SUBNET_1
  TF_VAR_SUBNET_2
  TF_VAR_SUBNET_3
  TF_VAR_SUBNET_4
  TF_VAR_FGT1_STATIC_IP_PORT1
  TF_VAR_FGT1_STATIC_IP_PORT2
  TF_VAR_FGT1_STATIC_IP_PORT3
  TF_VAR_FGT1_STATIC_IP_PORT4
  TF_VAR_FGT1_PORT4_MGMT_GATEWAY
  TF_VAR_FGT2_STATIC_IP_PORT1
  TF_VAR_FGT2_STATIC_IP_PORT2
  TF_VAR_FGT2_STATIC_IP_PORT3
  TF_VAR_FGT2_STATIC_IP_PORT4
  TF_VAR_FGT2_PORT4_MGMT_GATEWAY
  TF_VAR_SSH_PUBLIC_KEY
  TF_VAR_CATALOG_OFFERING_VERSION_CRN
  TF_VAR_CATALOG_OFFERING_PLAN_CRN
)
for var in "${REQUIRED_VARS[@]}"; do
  [ -z "${!var:-}" ] && echo "✗ Missing required variable: $var" && exit 1
done
echo "✓ All required variables present"

# Enable CI mode — opens inbound SSH (22) for acceptance tests
export TF_VAR_CI_MODE=true

# Lock CI inbound rules to this pipeline worker's public IP only (never 0.0.0.0/0)
PIPELINE_IP="$(curl -sf --max-time 10 https://checkip.amazonaws.com || curl -sf --max-time 10 https://api.ipify.org)"
if [[ -z "$PIPELINE_IP" ]]; then
  echo "✗ Could not determine pipeline worker public IP — cannot set CI_ALLOWED_CIDR"
  exit 1
fi
export TF_VAR_CI_ALLOWED_CIDR="${PIPELINE_IP}/32"
echo "✓ CI inbound rules locked to pipeline worker IP: $TF_VAR_CI_ALLOWED_CIDR"

# Pre-deploy cleanup: delete any orphaned VNIs and reserved IPs from previous failed runs.
echo "Checking for orphaned resources from previous failed runs..."
IAM_TOKEN="$(curl -sf --max-time 15 -X POST \
  "https://iam.cloud.ibm.com/identity/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=urn:ibm:params:oauth:grant-type:apikey&apikey=${IC_API_KEY}" \
  | jq -r '.access_token // empty' 2>/dev/null || true)"

if [[ -n "$IAM_TOKEN" ]]; then
  VPC_REGION="${TF_VAR_REGION:-us-south}"
  VPC_API="https://${VPC_REGION}.iaas.cloud.ibm.com/v1"

  VPC_ID="$(curl -sf --max-time 15 \
    "${VPC_API}/vpcs?name=${TF_VAR_VPC}&version=2024-01-01&generation=2" \
    -H "Authorization: Bearer ${IAM_TOKEN}" \
    | jq -r '.vpcs[0].id // empty' 2>/dev/null || true)"

  if [[ -n "$VPC_ID" ]]; then
    # ── Step 1: delete orphaned VNIs ─────────────────────────────────────────
    ORPHANED_VNIS="$(curl -sf --max-time 15 \
      "${VPC_API}/virtual_network_interfaces?version=2024-01-01&generation=2" \
      -H "Authorization: Bearer ${IAM_TOKEN}" \
      | jq -r --arg vpc "$VPC_ID" \
      '[.virtual_network_interfaces[] | select(.name | test("fortigate-terraform-fgt[12]-")) | select(.vpc.id == $vpc)] | .[].id' \
      2>/dev/null || true)"

    if [[ -n "$ORPHANED_VNIS" ]]; then
      echo "  Found orphaned VNIs — deleting:"
      echo "$ORPHANED_VNIS" | while read -r vni_id; do
        echo "    Deleting VNI: $vni_id"
        curl -sf --max-time 15 -X DELETE \
          "${VPC_API}/virtual_network_interfaces/${vni_id}?version=2024-01-01&generation=2" \
          -H "Authorization: Bearer ${IAM_TOKEN}" || true
      done
      echo "  Waiting 10s for VNI deletion to propagate..."
      sleep 10
    else
      echo "  No orphaned VNIs found"
    fi

    # ── Step 2: delete orphaned reserved IPs at our static addresses ─────────
    STATIC_IPS="${TF_VAR_FGT1_STATIC_IP_PORT1} ${TF_VAR_FGT1_STATIC_IP_PORT2} ${TF_VAR_FGT1_STATIC_IP_PORT3} ${TF_VAR_FGT1_STATIC_IP_PORT4} ${TF_VAR_FGT2_STATIC_IP_PORT1} ${TF_VAR_FGT2_STATIC_IP_PORT2} ${TF_VAR_FGT2_STATIC_IP_PORT3} ${TF_VAR_FGT2_STATIC_IP_PORT4}"
    for SUBNET_ID in "${TF_VAR_SUBNET_1}" "${TF_VAR_SUBNET_2}" "${TF_VAR_SUBNET_3}" "${TF_VAR_SUBNET_4}"; do
      [[ -z "$SUBNET_ID" ]] && continue
      RIPS="$(curl -sf --max-time 15 \
        "${VPC_API}/subnets/${SUBNET_ID}/reserved_ips?version=2024-01-01&generation=2" \
        -H "Authorization: Bearer ${IAM_TOKEN}" \
        | jq -r '.reserved_ips[] | select(.auto_delete == false) | "\(.id) \(.address)"' \
        2>/dev/null || true)"
      while IFS=' ' read -r rip_id rip_addr; do
        [[ -z "$rip_id" ]] && continue
        for static_ip in $STATIC_IPS; do
          if [[ "$rip_addr" == "$static_ip" ]]; then
            echo "  Deleting orphaned reserved IP $rip_id ($rip_addr) in subnet $SUBNET_ID"
            curl -sf --max-time 15 -X DELETE \
              "${VPC_API}/subnets/${SUBNET_ID}/reserved_ips/${rip_id}?version=2024-01-01&generation=2" \
              -H "Authorization: Bearer ${IAM_TOKEN}" || true
            break
          fi
        done
      done <<< "$RIPS"
    done
    echo "  Reserved IP cleanup complete"

  else
    echo "  Could not resolve VPC ID for '$TF_VAR_VPC' — skipping orphan cleanup"
  fi
else
  echo "  Could not obtain IAM token — skipping orphan cleanup (non-fatal)"
fi

rm -rf .terraform .terraform.lock.hcl
TF_PLUGIN_CACHE_DIR="" terraform init -no-color
terraform plan -no-color -out=tfplan

apply_failed=0
terraform apply -auto-approve -no-color -parallelism=5 tfplan || apply_failed=1

if [[ "$apply_failed" -eq 1 ]]; then
  echo "✗ terraform apply failed — running destroy to clean up partial resources"
  terraform destroy -auto-approve -no-color || true
  exit 1
fi

OUTPUT_FILE="${TF_OUTPUTS_PATH:-$TERRAFORM_DIR/terraform-outputs.json}"
terraform output -json > "$OUTPUT_FILE"
echo "✓ Deployment complete - outputs saved to $OUTPUT_FILE"

# Save state to WORKSPACE so destroy stage (separate container) can find the resources to remove
cp terraform.tfstate "$WORKSPACE/terraform.tfstate"
echo "✓ State saved to $WORKSPACE/terraform.tfstate"
