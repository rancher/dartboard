terraform {
  required_version = "> 1.6.2"
  required_providers {
    harvester = {
      source  = "harvester/harvester"
      version = "0.6.5"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "4.0.3"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "2.7.1"
    }
    ssh = {
      source  = "loafoe/ssh"
      version = "2.7.0"
    }
  }
}
