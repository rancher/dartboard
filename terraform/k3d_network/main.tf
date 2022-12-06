terraform {
  required_providers {
    docker = {
      source = "kreuzwerker/docker"
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
