terraform {
  required_version = "> 1.6.0"
  required_providers {
    ibm = {
      source  = "IBM-Cloud/ibm"
      version = "~> 2.1.0"
    }
  }
}

# Configure the IBM Provider
provider "ibm" {
  region           = var.REGION
  ibmcloud_api_key = var.IBMCLOUD_API_KEY
}
