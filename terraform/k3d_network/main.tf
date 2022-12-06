terraform {
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "2.16.0"
    }
  }
}

resource "docker_network" "network" {
  name   = "${var.project_name}-k3d"
  driver = "bridge"
}

output "name" {
  value = docker_network.network.name
}
