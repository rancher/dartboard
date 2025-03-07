terraform {
  required_version = "1.8.2"
  required_providers {
    harvester = {
      source  = "harvester/harvester"
      version = "0.6.6"
      # 0.6.5 does not currently have a Darwin binary built for it
      # locking to 0.6.4 until there is a newer version with a Darwin binary
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
