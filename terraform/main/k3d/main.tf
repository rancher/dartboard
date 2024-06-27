terraform {
  required_version = "1.6.2"
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "2.23.1"
    }
    k3d = {
      source  = "moio/k3d"
      version = "0.0.10"
    }
  }
}

module "network" {
  source       = "../../modules/k3d_network"
  project_name = local.project_name
}

module "cluster" {
  count          = length(local.clusters)
  source         = "../../modules/k3d_k3s"
  project_name   = local.project_name
  name           = local.clusters[count.index].name
  server_count   = local.clusters[count.index].server_count
  agent_count    = local.clusters[count.index].agent_count
  distro_version = local.clusters[count.index].distro_version

  sans                  = ["${local.clusters[count.index].name}.local.gd"]
  kubernetes_api_port   = local.first_kubernetes_api_port + count.index
  app_http_port         = local.first_app_http_port + count.index
  app_https_port        = local.first_app_https_port + count.index
  network_name          = module.network.name
  pull_proxy_registries = module.network.pull_proxy_registries
  enable_audit_log      = local.clusters[count.index].name == "upstream"
}
