# Your SSH key
variable "SSH_PUBLIC_KEY" {
  default     = ""
  description = "The name(ID) of your SSH public key to be used."
}
// Magic Value for Catalog Validation that initializes Terraform with a specific version.
// Only needed in IBM catalog.
variable "TF_VERSION" {
  default     = "1.6.6"
  description = "Terraform version to be used in validation."
}

// IBM Regions
variable "REGION" {
  type        = string
  default     = "us-east"
  description = "Deployment region."
}

// IBM availability zones
variable "ZONE" {
  type        = string
  default     = "us-east-1"
  description = "Deployment zone. Currently only a single zone is supported."
}

variable "VPC" {
  type        = string
  default     = ""
  description = "Name of the VPC you want to deploy a FortiGate into."
}

variable "SUBNET_1" {
  type        = string
  default     = ""
  description = "The ID of the Primary, Public Subnet Used for port1 on the FortiGate. A Public Gateway MUST be attached to allow the license on the secondary node to register."
}
variable "SUBNET_2" {
  type        = string
  default     = ""
  description = "The ID of the Secondary, Private Subnet Used for port2 on the FortiGate."
}
variable "SUBNET_3" {
  type        = string
  default     = ""
  description = "The ID of the Subnet for the HA heartbeat mechanism. Tied to Port3."
}
variable "SUBNET_4" {
  type        = string
  default     = ""
  description = "The ID of the Subnet used for the HA management subnet. Tied to Port4."
}

variable "NETMASK" {
  type        = string
  default     = "255.255.255.0"
  description = "Subnet mask for the static IP address and NIC of each FortiGate."
}

variable "FGT1_STATIC_IP_PORT1" {
  type        = string
  default     = ""
  description = "Static IP assignment for Port 1 of the Primary (ACTIVE) FortiGate."
}
variable "FGT1_STATIC_IP_PORT2" {
  type        = string
  default     = ""
  description = "Static IP assignment for Port 2 of the Primary (ACTIVE) FortiGate."
}
// Used for HA Heartbeat mechanism.
variable "FGT1_STATIC_IP_PORT3" {
  type        = string
  default     = ""
  description = "Port used for the HA Heartbeat mechanism."
}
// Used as the HA Management port.
variable "FGT1_STATIC_IP_PORT4" {
  type        = string
  default     = ""
  description = "HA management port."
}
variable "FGT1_PORT4_MGMT_GATEWAY" {
  type        = string
  default     = ""
  description = "Gateway for Port 4 (HA management port) on the primary (ACTIVE) FortiGate."
}
// FortiGate 2 (PASSIVE) PORTS
variable "FGT2_STATIC_IP_PORT1" {
  type        = string
  default     = ""
  description = "STATIC IP Assignment for Port 1 on the Secondary (PASSIVE) FortiGate."
}
variable "FGT2_STATIC_IP_PORT2" {
  type        = string
  default     = ""
  description = "STATIC IP Assignment for Port 2 on the Secondary (PASSIVE) FortiGate."
}
// Used for HA Heartbeat mechanism.
variable "FGT2_STATIC_IP_PORT3" {
  type        = string
  default     = ""
  description = "Port used for the HA Heartbeat mechanism."
}
// Used as the HA management port.
variable "FGT2_STATIC_IP_PORT4" {
  type        = string
  default     = ""
  description = "HA management port."
}
variable "FGT2_PORT4_MGMT_GATEWAY" {
  type        = string
  default     = ""
  description = "Gateway for Port 4 (HA management port) on the secondary (PASSIVE) FortiGate."
}

variable "RESOURCE_GRP" {
  type        = string
  default     = "Default"
  description = "The RESOURCE Group Name to attach to the FortiGate instances."
}

data "ibm_resource_group" "rg" {
  name = var.RESOURCE_GRP
}

// For easy identification, the name of the cluster uses the format cluster-name-resource-randomSuffix."
// Name must be lowercase.
variable "CLUSTER_NAME" {
  type        = string
  default     = "fortigate-terraform"
  description = "Name of the cluster (must be lowercase). For easy identification, the format cluster-name-resource-randomSuffix is used."
}
// Random suffix to avoid cluster name collisions and help identify the  cluster.
resource "random_string" "random_suffix" {
  length           = 4
  special          = true
  override_special = ""
  min_lower        = 4
}

// ID of the public gateway in the same zone to attach to the port1 subnet.
// A public gateway must be pre-created and attached to SUBNET_1 before deployment.
variable "PUBLIC_GATEWAY_ID" {
  type        = string
  description = "ID of the public gateway to attach to the port1 subnet. A public gateway must be attached to the port1 subnet to provide outbound internet access, which is required for the secondary FortiGate node to register its license."
}

// Specify the Version CRN for the Catalog offering representing the Licensed Fortinet HA Firewall.
variable "CATALOG_OFFERING_VERSION_CRN" {
  default     = ""
  description = "IBM Cloud Licensed Firewall Version CRN"
}

// Specify the Plan CRN for the Catalog offering representing the Licensed Fortinet HA Firewall.
variable "CATALOG_OFFERING_PLAN_CRN" {
  default     = ""
  description = "IBM Cloud Licensed Firewall Plan CRN"
}

variable "IBMCLOUD_API_KEY" {
  default     = ""
  description = "Your IBM USER API key. Refer to the README for links to documentation for IBM API keys. This value is required for the SDN Connector for HA SYNC."
}

variable "CI_MODE" {
  type        = bool
  default     = false
  description = "When true, adds inbound SSH (22) and HTTPS (443) rules to public security group for CI acceptance testing."
}

variable "CI_ALLOWED_CIDR" {
  type        = string
  default     = ""
  description = "CIDR block allowed to reach the FortiGate on ports 22 and 443 during CI. Set to the pipeline worker's public IP as a /32 (e.g. '169.46.1.2/32'). Only used when CI_MODE=true."
}
