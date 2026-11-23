terraform {
  required_version = ">=1.8.2"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "5.14.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "4.4.2"
    }
    ssh = {
      source  = "loafoe/ssh"
      version = "2.9.0"
    }
  }
}
