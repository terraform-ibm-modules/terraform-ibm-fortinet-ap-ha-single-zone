#!/usr/bin/env bash
# CI environment loader for FortiGate A/P HA Single Zone
# Source this file — do NOT execute it directly.
# Sets all TF_VAR_* exports and IC_API_KEY needed by ci-deploy.sh and ci-destroy.sh.
#
# Required pipeline properties:
#   TF_CI_CONFIG          — JSON blob with all non-secret deployment config (get_env)
#   TF_VAR_IBMCLOUD_API_KEY — secure property (get_secret)

# Pull all non-secret config from the single JSON pipeline property TF_CI_CONFIG.
TF_CI_CONFIG="$(get_env TF_CI_CONFIG "")"
if [[ -z "$TF_CI_CONFIG" ]]; then
  echo "✗ Missing required pipeline property: TF_CI_CONFIG (JSON blob with all deployment config)"
  exit 1
fi

_jq() { echo "$TF_CI_CONFIG" | jq -r ".$1 // empty"; }
export TF_VAR_REGION="${TF_VAR_REGION:-$(_jq REGION)}"
export TF_VAR_ZONE="${TF_VAR_ZONE:-$(_jq ZONE)}"
export TF_VAR_VPC="${TF_VAR_VPC:-$(_jq VPC)}"
export TF_VAR_SUBNET_1="${TF_VAR_SUBNET_1:-$(_jq SUBNET_1)}"
export TF_VAR_SUBNET_2="${TF_VAR_SUBNET_2:-$(_jq SUBNET_2)}"
export TF_VAR_SUBNET_3="${TF_VAR_SUBNET_3:-$(_jq SUBNET_3)}"
export TF_VAR_SUBNET_4="${TF_VAR_SUBNET_4:-$(_jq SUBNET_4)}"
export TF_VAR_NETMASK="${TF_VAR_NETMASK:-$(_jq NETMASK)}"
export TF_VAR_FGT1_STATIC_IP_PORT1="${TF_VAR_FGT1_STATIC_IP_PORT1:-$(_jq FGT1_STATIC_IP_PORT1)}"
export TF_VAR_FGT1_STATIC_IP_PORT2="${TF_VAR_FGT1_STATIC_IP_PORT2:-$(_jq FGT1_STATIC_IP_PORT2)}"
export TF_VAR_FGT1_STATIC_IP_PORT3="${TF_VAR_FGT1_STATIC_IP_PORT3:-$(_jq FGT1_STATIC_IP_PORT3)}"
export TF_VAR_FGT1_STATIC_IP_PORT4="${TF_VAR_FGT1_STATIC_IP_PORT4:-$(_jq FGT1_STATIC_IP_PORT4)}"
export TF_VAR_FGT1_PORT4_MGMT_GATEWAY="${TF_VAR_FGT1_PORT4_MGMT_GATEWAY:-$(_jq FGT1_PORT4_MGMT_GATEWAY)}"
export TF_VAR_FGT2_STATIC_IP_PORT1="${TF_VAR_FGT2_STATIC_IP_PORT1:-$(_jq FGT2_STATIC_IP_PORT1)}"
export TF_VAR_FGT2_STATIC_IP_PORT2="${TF_VAR_FGT2_STATIC_IP_PORT2:-$(_jq FGT2_STATIC_IP_PORT2)}"
export TF_VAR_FGT2_STATIC_IP_PORT3="${TF_VAR_FGT2_STATIC_IP_PORT3:-$(_jq FGT2_STATIC_IP_PORT3)}"
export TF_VAR_FGT2_STATIC_IP_PORT4="${TF_VAR_FGT2_STATIC_IP_PORT4:-$(_jq FGT2_STATIC_IP_PORT4)}"
export TF_VAR_FGT2_PORT4_MGMT_GATEWAY="${TF_VAR_FGT2_PORT4_MGMT_GATEWAY:-$(_jq FGT2_PORT4_MGMT_GATEWAY)}"
export TF_VAR_PUBLIC_GATEWAY_ID="${TF_VAR_PUBLIC_GATEWAY_ID:-$(_jq PUBLIC_GATEWAY_ID)}"
export TF_VAR_RESOURCE_GRP="${TF_VAR_RESOURCE_GRP:-$(_jq RESOURCE_GRP)}"
export TF_VAR_SSH_PUBLIC_KEY="${TF_VAR_SSH_PUBLIC_KEY:-$(_jq SSH_PUBLIC_KEY)}"
export TF_VAR_CATALOG_OFFERING_VERSION_CRN="${TF_VAR_CATALOG_OFFERING_VERSION_CRN:-$(_jq CATALOG_OFFERING_VERSION_CRN)}"
export TF_VAR_CATALOG_OFFERING_PLAN_CRN="${TF_VAR_CATALOG_OFFERING_PLAN_CRN:-$(_jq CATALOG_OFFERING_PLAN_CRN)}"

# Secure property — must use get_secret, not get_env
export TF_VAR_IBMCLOUD_API_KEY="${TF_VAR_IBMCLOUD_API_KEY:-$(get_secret TF_VAR_IBMCLOUD_API_KEY "")}"

# Override IC_API_KEY so the IBM Terraform provider uses the deployment key,
# not the OnePipeline service ID key injected into the container.
export IC_API_KEY="$TF_VAR_IBMCLOUD_API_KEY"
