terraform {
  required_version = "1.3.7"
  required_providers {
    docker = {
      source  = "kreuzwerker/docker"
      version = "2.23.1"
    }
    k3d = {
      source  = "moio/k3d"
      version = "0.0.7"
    }
  }
}

module "network" {
  source       = "./k3d_network"
  project_name = local.project_name
}

module "cluster" {
  for_each                 = local.clusters
  source                   = "./k3d_k3s"
  project_name             = local.project_name
  name                     = each.key
  network_name             = module.network.name
  server_count             = each.value.server_count
  agent_count              = each.value.agent_count
  agent_labels             = each.value.agent_labels
  agent_taints             = each.value.agent_taints
  distro_version           = each.value.distro_version
  sans                     = [each.value.san]
  kubernetes_api_port      = each.value.kubernetes_api_port
  additional_port_mappings = [[each.value.public_http_port, 80], [each.value.public_https_port, 443]]
}
