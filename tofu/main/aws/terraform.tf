terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "6.9.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "4.1.0"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "3.0.2"
    }
    ssh = {
      source  = "loafoe/ssh"
      version = "2.7.0"
    }
  }
}
