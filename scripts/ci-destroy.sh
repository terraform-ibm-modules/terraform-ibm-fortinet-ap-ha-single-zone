#!/usr/bin/env bash
# CI Pipeline - Destroy Script for FortiGate A/P HA Single Zone
# Sourced by terraform-ci-acceptance-test.sh after tests complete (pass or fail)
# TERRAFORM_DIR is set by the pipeline before sourcing this script

set -euo pipefail

echo "============================================"
echo "FortiGate HA Single Zone - Destroy"
echo "============================================"

cd "$TERRAFORM_DIR"

# Load all TF_VAR_* exports and IC_API_KEY from the shared env loader.
# This script runs in a separate container from the deploy stage so env vars
# are not inherited — ci-env.sh re-reads them from the pipeline env store.
# shellcheck source=scripts/ci-env.sh
source "${TERRAFORM_DIR}/scripts/ci-env.sh"

# Re-initialise providers — wipe any stale cache first to prevent checksum mismatches
rm -rf .terraform .terraform.lock.hcl
TF_PLUGIN_CACHE_DIR="" terraform init -no-color

# Restore state from WORKSPACE so destroy knows which resources to remove
if [[ -f "$WORKSPACE/terraform.tfstate" ]]; then
  cp "$WORKSPACE/terraform.tfstate" terraform.tfstate
  echo "✓ State restored from $WORKSPACE/terraform.tfstate"
else
  echo "⚠ No state file found in WORKSPACE — destroy may not remove all resources"
fi

terraform destroy -auto-approve -no-color
echo "✓ Resources destroyed"

# Post-destroy VNI cleanup: VNIs have auto_delete=false so IBM Cloud does not
# remove them when the VSI is destroyed. Delete any VNIs still attached to our
# CI subnets — filtered by subnet ID so we never touch resources belonging to
# other deployments that share the VPC.
echo "Post-destroy VNI cleanup..."
IAM_TOKEN="$(curl -sf --max-time 15 -X POST \
  "https://iam.cloud.ibm.com/identity/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=urn:ibm:params:oauth:grant-type:apikey&apikey=${IC_API_KEY}" \
  | jq -r '.access_token // empty' 2>/dev/null || true)"

if [[ -n "$IAM_TOKEN" ]]; then
  VPC_REGION="${TF_VAR_REGION:-us-south}"
  VPC_API="https://${VPC_REGION}.iaas.cloud.ibm.com/v1"

  for SUBNET_ID in "${TF_VAR_SUBNET_1}" "${TF_VAR_SUBNET_2}" "${TF_VAR_SUBNET_3}" "${TF_VAR_SUBNET_4}"; do
    [[ -z "$SUBNET_ID" ]] && continue
    SUBNET_VNIS="$(curl -sf --max-time 15 \
      "${VPC_API}/virtual_network_interfaces?version=2024-01-01&generation=2" \
      -H "Authorization: Bearer ${IAM_TOKEN}" \
      | jq -r --arg subnet "$SUBNET_ID" \
      '[.virtual_network_interfaces[] | select(.subnet.id == $subnet)] | .[].id' \
      2>/dev/null || true)"
    if [[ -n "$SUBNET_VNIS" ]]; then
      echo "  Found VNIs in subnet $SUBNET_ID — deleting:"
      echo "$SUBNET_VNIS" | while read -r vni_id; do
        echo "    Deleting VNI: $vni_id"
        curl -sf --max-time 15 -X DELETE \
          "${VPC_API}/virtual_network_interfaces/${vni_id}?version=2024-01-01&generation=2" \
          -H "Authorization: Bearer ${IAM_TOKEN}" || true
      done
    else
      echo "  No VNIs found in subnet $SUBNET_ID"
    fi
  done
  echo "  VNI cleanup complete"
else
  echo "  Could not obtain IAM token — skipping VNI cleanup (non-fatal)"
fi
