output "FortiGate_Public_IP" {
  value = ibm_is_floating_ip.publicip.address
}
output "FGT1_Public_HA_Management_IP" {
  value = ibm_is_floating_ip.publicip2.address
}
output "FGT2_Public_HA_Management_IP" {
  value = ibm_is_floating_ip.publicip3.address
}
output "Security_Group_Public_ID" {
  description = "ID of the public security group (port1 + port4)"
  value       = ibm_is_security_group.fgt_sg_public.id
}
output "Security_Group_Public_Name" {
  description = "Name of the public security group (port1 + port4)"
  value       = ibm_is_security_group.fgt_sg_public.name
}
output "Security_Group_Private_ID" {
  description = "ID of the private security group (port2 + port3)"
  value       = ibm_is_security_group.fgt_sg_private.id
}
output "Security_Group_Private_Name" {
  description = "Name of the private security group (port2 + port3)"
  value       = ibm_is_security_group.fgt_sg_private.name
}

output "Port1_Public_Gateway_ID" {
  description = "ID of the public gateway attached to port1 subnet (existing or newly created)"
  value       = local.resolved_public_gateway_id
}

output "Selected_VSI_Profile" {
  description = "The VSI profile automatically selected based on the catalog offering plan CRN"
  value       = local.selected_profile
}

output "Catalog_Offering_Version_CRN" {
  description = "The catalog offering version CRN used for deployment"
  value       = var.CATALOG_OFFERING_VERSION_CRN
}

output "Catalog_Offering_Plan_CRN" {
  description = "The catalog offering plan CRN used for deployment"
  value       = var.CATALOG_OFFERING_PLAN_CRN
}

output "Username" {
  value = "admin"
}

output "FGT1_Default_Admin_Password" {
  description = "Default admin password for FortiGate 1 (instance ID). Password may be empty on first boot."
  value       = ibm_is_instance.fgt1.id
}

output "FGT2_Default_Admin_Password" {
  description = "Default admin password for FortiGate 2 (instance ID). Password may be empty on first boot."
  value       = ibm_is_instance.fgt2.id
}