terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "3.9.0"
    }
    k3d = {
      source  = "moio/k3d"
      version = "0.0.12"
    }
  }
}
