terraform {
  required_providers {
    k3d = {
      source = "moio/k3d"
    }
    docker = {
      source = "kreuzwerker/docker"
    }
  }
}

resource "docker_network" "network" {
  name   = "${var.project_name}-k3d"
  driver = "bridge"
}

resource "k3d_registry" "docker_io_proxy" {
  name = "${var.project_name}-docker-io-proxy"
  port {
    host_port = "5001"
  }
  proxy_remote_url = "https://registry-1.docker.io"
  network          = docker_network.network.name

  volume {
    source      = var.docker_io_proxy_directory
    destination = "/var/lib/registry"
  }
}
