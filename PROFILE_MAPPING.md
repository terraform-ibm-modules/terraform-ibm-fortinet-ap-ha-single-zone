# VSI Profile Mapping for Licensed Fortinet Firewall HA Single-Zone

## Overview

This Terraform configuration automatically maps IBM Cloud Catalog Plan CRNs to the appropriate VSI profiles based on the licensing tier selected by the user for the HA single-zone deployment.

## How It Works

1. **User Input**: The user specifies a `CATALOG_OFFERING_PLAN_CRN` variable when deploying the Fortinet firewall HA cluster.

2. **Automatic Mapping**: The Terraform configuration looks up the corresponding VSI profile from a predefined mapping in `locals.tf`.

3. **Instance Creation**: Both FortiGate instances (active and passive) are created with the correct profile that matches the licensing tier.

4. **Region-Specific Overrides**: For regions where certain profiles are unavailable (e.g., Chennai), the configuration automatically selects alternative profiles.

## Supported Plan CRNs and Profiles

Based on `Licensed_Firewall_Profiles_May_2026(Fortinet vFSA).csv`:

### Enterprise License
| vCPU | Segment | VSI Profile | Plan CRN |
|------|---------|-------------|----------|
| 8    | Medium  | cx3d-8x20   | crn:v1:bluemix:public:globalcatalog-collection:global:a/9fb39c8182db4358880b1ef4a03d3506:f57a0d21-54f2-4439-83d6-181e19f342e9:plan:sw.f57a0d21-54f2-4439-83d6-181e19f342e9.c9e40066-91d5-470b-b048-d900b79be72f |
| 16   | Large   | cx3d-16x40  | crn:v1:bluemix:public:globalcatalog-collection:global:a/9fb39c8182db4358880b1ef4a03d3506:f57a0d21-54f2-4439-83d6-181e19f342e9:plan:sw.f57a0d21-54f2-4439-83d6-181e19f342e9.c332973d-f363-4223-b257-a672ced98996 |
| 32   | X-large | cx3d-32x80  | crn:v1:bluemix:public:globalcatalog-collection:global:a/9fb39c8182db4358880b1ef4a03d3506:f57a0d21-54f2-4439-83d6-181e19f342e9:plan:sw.f57a0d21-54f2-4439-83d6-181e19f342e9.e677cc20-3263-445c-aa57-19822204471f |

### UTP (Unified Threat Protection) License
| vCPU | Segment | VSI Profile | Plan CRN |
|------|---------|-------------|----------|
| 2    | Small   | cx3d-2x5    | crn:v1:bluemix:public:globalcatalog-collection:global:a/9fb39c8182db4358880b1ef4a03d3506:f57a0d21-54f2-4439-83d6-181e19f342e9:plan:sw.f57a0d21-54f2-4439-83d6-181e19f342e9.8ce68776-289b-4dc5-84d1-845f7290f22e |
| 8    | Medium  | cx3d-8x20   | crn:v1:bluemix:public:globalcatalog-collection:global:a/9fb39c8182db4358880b1ef4a03d3506:f57a0d21-54f2-4439-83d6-181e19f342e9:plan:sw.f57a0d21-54f2-4439-83d6-181e19f342e9.8aca8249-18f1-4da7-bfb7-78fcccc1ed9e |
| 16   | Large   | cx3d-16x40  | crn:v1:bluemix:public:globalcatalog-collection:global:a/9fb39c8182db4358880b1ef4a03d3506:f57a0d21-54f2-4439-83d6-181e19f342e9:plan:sw.f57a0d21-54f2-4439-83d6-181e19f342e9.419fa63c-e8f5-4065-b676-68cbc11af3e3 |

### ATP (Advanced Threat Protection) License
| vCPU | Segment | VSI Profile | Plan CRN |
|------|---------|-------------|----------|
| 2    | Small   | cx2-2x4     | crn:v1:bluemix:public:globalcatalog-collection:global:a/9fb39c8182db4358880b1ef4a03d3506:f57a0d21-54f2-4439-83d6-181e19f342e9:plan:sw.f57a0d21-54f2-4439-83d6-181e19f342e9.23def19e-5fea-4259-8bb3-09325c72a75f |
| 8    | Medium  | cx2-8x16    | crn:v1:bluemix:public:globalcatalog-collection:global:a/9fb39c8182db4358880b1ef4a03d3506:f57a0d21-54f2-4439-83d6-181e19f342e9:plan:sw.f57a0d21-54f2-4439-83d6-181e19f342e9.c17d7796-0092-47e6-8d58-48533fba9aa5 |

### Test License
| vCPU | Segment | VSI Profile | Plan CRN |
|------|---------|-------------|----------|
| 2    | UTP TEST | cx2-2x4    | crn:v1:bluemix:public:globalcatalog-collection:global:a/9fb39c8182db4358880b1ef4a03d3506:f57a0d21-54f2-4439-83d6-181e19f342e9:plan:sw.f57a0d21-54f2-4439-83d6-181e19f342e9.2b990678-903c-4f3f-ad2a-1b789fc71619 |

## Region-Specific Profile Overrides

Some regions don't support all profile types. The configuration automatically handles these cases:

### Chennai (in-che) Region
- **ATP 2vCPU Small**: Uses `cx3d-2x5` instead of `cx2-2x4`
- **ATP 8vCPU Medium**: Uses `cx3d-8x20` instead of `cx2-8x16`
- **UTP 2vCPU TEST**: Uses `cx3d-2x5` instead of `cx2-2x4`

## Usage Example

```hcl
# Specify the plan CRN for Enterprise 8vCPU Medium
CATALOG_OFFERING_PLAN_CRN = "crn:v1:bluemix:public:globalcatalog-collection:global:a/9fb39c8182db4358880b1ef4a03d3506:f57a0d21-54f2-4439-83d6-181e19f342e9:plan:sw.f57a0d21-54f2-4439-83d6-181e19f342e9.c9e40066-91d5-470b-b048-d900b79be72f"

# The profile will automatically be set to "cx3d-8x20" for both FortiGate instances
# No need to manually specify the PROFILE variable
```

## Validation

The `CATALOG_OFFERING_PLAN_CRN` is **required** and must be one of the supported plan CRNs listed above. The configuration includes validation logic that checks if the provided plan CRN exists in the supported plans list.

If an invalid or missing plan CRN is provided, Terraform will fail with a clear error message during the plan phase:
```
ERROR: Invalid or missing plan_crn. The CATALOG_OFFERING_PLAN_CRN must be one of the supported plan CRNs. See PROFILE_MAPPING.md for the list of valid CRNs.
```
