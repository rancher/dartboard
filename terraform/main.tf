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

module "old_upstream_cluster" {
  source                   = "./k3d_k3s"
  project_name             = local.project_name
  name                     = "oldupstream"
  network_name             = module.network.name
  server_count             = local.old_upstream_server_count
  agent_count              = local.old_upstream_agent_count
  distro_version           = local.distro_version
  sans                     = [local.old_upstream_san]
  kubernetes_api_port      = local.old_upstream_kubernetes_api_port
  additional_port_mappings = [[local.old_upstream_public_port, 443]]
}

module "old_downstream_cluster" {
  source              = "./k3d_k3s"
  project_name        = local.project_name
  name                = "olddownstream"
  network_name        = module.network.name
  server_count        = local.old_downstream_server_count
  agent_count         = local.old_downstream_agent_count
  distro_version      = local.distro_version
  sans                = [local.old_downstream_san]
  kubernetes_api_port = local.old_downstream_kubernetes_api_port
}

module "new_upstream_cluster" {
  source                   = "./k3d_k3s"
  project_name             = local.project_name
  name                     = "newupstream"
  network_name             = module.network.name
  server_count             = local.new_upstream_server_count
  agent_count              = local.new_upstream_agent_count
  distro_version           = local.distro_version
  sans                     = [local.new_upstream_san]
  kubernetes_api_port      = local.new_upstream_kubernetes_api_port
  additional_port_mappings = [[local.new_upstream_public_port, 443]]
}

module "new_downstream_cluster" {
  source              = "./k3d_k3s"
  project_name        = local.project_name
  name                = "newdownstream"
  network_name        = module.network.name
  server_count        = local.new_downstream_server_count
  agent_count         = local.new_downstream_agent_count
  distro_version      = local.distro_version
  sans                = [local.new_downstream_san]
  kubernetes_api_port = local.new_downstream_kubernetes_api_port
}
