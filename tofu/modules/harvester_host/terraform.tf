terraform {
  required_version = "> 1.6.2"
  required_providers {
    harvester = {
      source  = "harvester/harvester"
      version = "0.6.4"
    }
    ssh = {
      source  = "loafoe/ssh"
      version = "2.7.0"
    }
  }
}
