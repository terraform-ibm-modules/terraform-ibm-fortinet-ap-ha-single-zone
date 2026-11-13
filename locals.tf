# Mapping of Partner Center Pricing Plan CRNs to VSI Profiles
# Based on Licensed_Firewall_Profiles_May_2026(Fortinet vFSA).csv

locals {
  # Default map of plan_crn to VSI profile (used for most regions)
  plan_crn_to_profile = {
    # Enterprise 8vCPU Medium
    "crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.c9e40066-91d5-470b-b048-d900b79be72f-global" = "cx3d-8x20"

    # Enterprise 16vCPU Large
    "crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.c332973d-f363-4223-b257-a672ced98996-global" = "cx3d-16x40"

    # Enterprise 32vCPU X-large
    "crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.e677cc20-3263-445c-aa57-19822204471f-global" = "cx3d-32x80"

    # UTP 2vCPU Small
    "crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.8ce68776-289b-4dc5-84d1-845f7290f22e-global" = "cx3d-2x5"

    # UTP 8vCPU Medium
    "crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.8aca8249-18f1-4da7-bfb7-78fcccc1ed9e-global" = "cx3d-8x20"

    # UTP 16vCPU Large
    "crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.419fa63c-e8f5-4065-b676-68cbc11af3e3-global" = "cx3d-16x40"

    # ATP 2vCPU Small
    "crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.23def19e-5fea-4259-8bb3-09325c72a75f-global" = "cx2-2x4"

    # ATP 8vCPU Medium
    "crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.c17d7796-0092-47e6-8d58-48533fba9aa5-global" = "cx2-8x16"
  }

  # Region-specific profile mappings for regions where certain profiles are unavailable
  # Key format: "region:plan_crn"
  region_specific_profile_overrides = {
    # Chennai (in-che), Mumbai (in-mum) and Montreal (ca-mon) region overrides - cx2 profiles not available
    # ATP 2vCPU Small - use cx3d-2x5 instead of cx2-2x4
    "in-che:crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.23def19e-5fea-4259-8bb3-09325c72a75f-global" = "cx3d-2x5"
    "in-mum:crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.23def19e-5fea-4259-8bb3-09325c72a75f-global" = "cx3d-2x5"
    "ca-mon:crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.23def19e-5fea-4259-8bb3-09325c72a75f-global" = "cx3d-2x5"

    # ATP 8vCPU Medium - use cx3d-8x20 instead of cx2-8x16
    "in-che:crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.c17d7796-0092-47e6-8d58-48533fba9aa5-global" = "cx3d-8x20"
    "in-mum:crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.c17d7796-0092-47e6-8d58-48533fba9aa5-global" = "cx3d-8x20"
    "ca-mon:crn:v1:bluemix:public:globalcatalog-collection:global::1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc:plan:sw.1082e7d2-5e2f-0a11-a3bc-f88a8e1931fc.c17d7796-0092-47e6-8d58-48533fba9aa5-global" = "cx3d-8x20"

    # Add more region-specific overrides here as needed
    # Format: "region-code:plan_crn" = "alternative_profile"
  }

  # Construct the lookup key for region-specific mapping
  region_profile_key = "${var.REGION}:${var.CATALOG_OFFERING_PLAN_CRN}"

  # Lookup the profile based on the plan_crn with region-specific override support
  # Priority: 1) Region-specific override, 2) Default mapping
  # The plan_crn must be provided and must be in the supported list
  selected_profile = (
    contains(keys(local.region_specific_profile_overrides), local.region_profile_key)
    ? local.region_specific_profile_overrides[local.region_profile_key]
    : local.plan_crn_to_profile[var.CATALOG_OFFERING_PLAN_CRN]
  )

  # Derive port1 subnet gateway — IBM VPC always assigns the first usable IP (.1) as gateway
  port1_gateway = cidrhost(data.ibm_is_subnet.subnet1.ipv4_cidr_block, 1)

  # True if SUBNET_1 already has a public gateway attached (detected from the subnet data source)
  subnet1_has_gateway = data.ibm_is_subnet.subnet1.public_gateway != ""

  # True when we need to create a new gateway:
  # - No explicit PUBLIC_GATEWAY_ID provided, AND
  # - The subnet does not already have one attached
  need_new_public_gateway = var.PUBLIC_GATEWAY_ID == "" && !local.subnet1_has_gateway

  # Resolve the gateway ID to use for attachment:
  # Priority: 1) explicit PUBLIC_GATEWAY_ID, 2) existing on subnet, 3) newly created
  resolved_public_gateway_id = (
    var.PUBLIC_GATEWAY_ID != ""
    ? var.PUBLIC_GATEWAY_ID
    : local.subnet1_has_gateway
    ? data.ibm_is_subnet.subnet1.public_gateway
    : ibm_is_public_gateway.port1_gateway[0].id
  )

}