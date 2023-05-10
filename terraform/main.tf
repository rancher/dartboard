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

module "upstream_cluster" {
  source                   = "./k3d_k3s"
  project_name             = local.project_name
  name                     = "upstream"
  network_name             = module.network.name
  server_count             = local.upstream_server_count
  agent_count              = local.upstream_agent_count
  labels                   = [{ key : "monitoring", value : "true", node_filters : ["agent:0"] }]
  taints                   = [{ key : "monitoring", value : "true", effect : "NoSchedule", node_filters : ["agent:0"] }]
  distro_version           = local.upstream_distro_version
  sans                     = [local.upstream_san]
  kubernetes_api_port      = local.upstream_kubernetes_api_port
  additional_port_mappings = [[local.upstream_public_port, 443]]
}

module "downstream_cluster" {
  source              = "./k3d_k3s"
  project_name        = local.project_name
  name                = "downstream"
  network_name        = module.network.name
  server_count        = local.downstream_server_count
  agent_count         = local.downstream_agent_count
  distro_version      = local.downstream_distro_version
  sans                = [local.downstream_san]
  kubernetes_api_port = local.downstream_kubernetes_api_port
}
