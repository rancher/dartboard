terraform {
  required_version = "1.3.1"
  required_providers {
    helm = {
      source  = "hashicorp/helm"
      version = "2.7.1"
    }
    docker = {
      source  = "kreuzwerker/docker"
      version = "2.23.1"
    }
    k3d = {
      source  = "pvotal-tech/k3d"
      version = "0.0.6"
    }
  }
}

provider "docker" {
  host = "unix:///var/run/docker.sock"
}

module "network" {
  source       = "./k3d_network"
  project_name = local.project_name
}

module "upstream_cluster" {
  source         = "./k3d_k3s"
  project_name   = local.project_name
  name           = "upstream"
  network_name   = module.network.name
  server_count   = local.upstream_server_count
  agent_count    = local.upstream_agent_count
  distro_version = local.upstream_distro_version
  sans           = [local.upstream_san]
  datastore      = var.upstream_datastore
  kine_image     = var.kine_image
}
