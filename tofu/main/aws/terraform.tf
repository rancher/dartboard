terraform {
  required_version = ">=1.8.2"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "6.42.0"
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
