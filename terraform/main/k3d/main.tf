terraform {
  required_version = "1.5.3"
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
  agent_labels   = local.clusters[count.index].agent_labels
  agent_taints   = local.clusters[count.index].agent_taints
  distro_version = local.clusters[count.index].distro_version

  sans                      = [local.clusters[count.index].local_name]
  local_kubernetes_api_port = local.first_local_kubernetes_api_port + count.index
  local_http_port           = local.first_local_http_port + count.index
  local_https_port          = local.first_local_https_port + count.index
  network_name              = module.network.name
  registry                  = module.network.registry
}
