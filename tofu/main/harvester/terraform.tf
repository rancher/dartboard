terraform {
  required_version = ">=1.8.2"
  required_providers {
    harvester = {
      source  = "harvester/harvester"
      version = "0.6.6"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "4.2.1"
    }
    ssh = {
      source  = "loafoe/ssh"
      version = "2.7.0"
    }
  }
}
