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

resource "k3d_registry" "proxy" {
  count = length(var.registry_pull_proxies)

  name = "${var.project_name}-${replace(var.registry_pull_proxies[count.index].name, ".", "-")}-proxy"
  port {
    host_port = var.first_proxy_port + count.index
  }
  proxy_remote_url = var.registry_pull_proxies[count.index].url
  network          = docker_network.network.name

  volume {
    source      = "/tmp/k3d-${replace(var.registry_pull_proxies[count.index].name, ".", "-")}-proxy"
    destination = "/var/lib/registry"
  }
}
