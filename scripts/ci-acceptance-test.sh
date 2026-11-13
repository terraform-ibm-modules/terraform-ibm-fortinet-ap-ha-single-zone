#!/usr/bin/env bash
# CI Pipeline - Acceptance Test Script for FortiGate A/P HA Single Zone
# Sourced by terraform-ci-acceptance-test.sh in the acceptance-test stage
# TERRAFORM_DIR and TF_OUTPUTS_PATH are set by the pipeline before sourcing

set -euo pipefail

echo "============================================"
echo "FortiGate HA Single Zone - Acceptance Tests"
echo "============================================"

cd "$TERRAFORM_DIR"

# Verify outputs file exists from deploy stage
TF_OUTPUTS_PATH="${TF_OUTPUTS_PATH:-$TERRAFORM_DIR/terraform-outputs.json}"
if [ ! -f "$TF_OUTPUTS_PATH" ]; then
  echo "✗ Terraform outputs not found at $TF_OUTPUTS_PATH"
  echo "  Deploy stage may have failed — check deploy stage logs"
  exit 1
fi
echo "✓ Terraform outputs found at $TF_OUTPUTS_PATH"

# Load IC_API_KEY and TF_VAR_REGION from the shared env loader
# shellcheck source=scripts/ci-env.sh
source "${TERRAFORM_DIR}/scripts/ci-env.sh"

# ── Update SSH SG rule to this acceptance-test container's public IP ──────────
# The deploy stage locked the inbound SSH/22 rule to the deploy worker's IP.
# This container runs on a different worker — patch the rule to our IP now.
TEST_WORKER_IP="$(curl -sf --max-time 10 https://checkip.amazonaws.com || curl -sf --max-time 10 https://api.ipify.org)"
if [[ -z "$TEST_WORKER_IP" ]]; then
  echo "✗ Could not determine acceptance-test worker public IP"
  exit 1
fi
TEST_WORKER_CIDR="${TEST_WORKER_IP}/32"
echo "✓ CI inbound rules locked to pipeline worker IP: $TEST_WORKER_CIDR"

VPC_REGION="${TF_VAR_REGION:-us-south}"
VPC_API="https://${VPC_REGION}.iaas.cloud.ibm.com/v1"
SG_ID="$(jq -r '.Security_Group_Public_ID.value // empty' "$TF_OUTPUTS_PATH")"

if [[ -z "$SG_ID" ]]; then
  echo "✗ Security_Group_Public_ID not found in $TF_OUTPUTS_PATH"
  exit 1
fi

IAM_TOKEN="$(curl -sf --max-time 15 -X POST \
  "https://iam.cloud.ibm.com/identity/token" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "grant_type=urn:ibm:params:oauth:grant-type:apikey&apikey=${IC_API_KEY}" \
  | jq -r '.access_token // empty')"

if [[ -z "$IAM_TOKEN" || "$IAM_TOKEN" == "null" ]]; then
  echo "✗ Failed to obtain IAM token — check IC_API_KEY"
  exit 1
fi

RULE_ID="$(curl -sf --max-time 15 \
  "${VPC_API}/security_groups/${SG_ID}/rules?version=2024-01-01&generation=2" \
  -H "Authorization: Bearer ${IAM_TOKEN}" \
  | jq -r '.rules[] | select(.direction=="inbound" and .protocol=="tcp" and .port_min==22 and .port_max==22) | .id' \
  | head -1)"

if [[ -z "$RULE_ID" ]]; then
  echo "✗ Could not find inbound TCP/22 rule in SG $SG_ID"
  exit 1
fi

PATCH_RESPONSE="$(curl -sf --max-time 15 -X PATCH \
  "${VPC_API}/security_groups/${SG_ID}/rules/${RULE_ID}?version=2024-01-01&generation=2" \
  -H "Authorization: Bearer ${IAM_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"remote\":{\"cidr_block\":\"${TEST_WORKER_CIDR}\"}}")"

if [[ -z "$PATCH_RESPONSE" ]]; then
  echo "✗ PATCH SG rule failed — empty response (check permissions or CIDR)"
  exit 1
fi
echo "  SG rule PATCH response: $PATCH_RESPONSE"
echo "✓ SSH SG rule updated — inbound TCP/22 now allows: $TEST_WORKER_CIDR"

# ── Run acceptance tests ──────────────────────────────────────────────────────
# CI_MODE=true enables acceptance tests in acceptance_test.go
export CI_MODE=true
export TF_OUTPUTS_PATH

cd "$TERRAFORM_DIR/tests"
echo "Running acceptance tests from: $(pwd)"

go test -v -timeout 60m -run "TestFortiGate|TestCatalogOffering"
