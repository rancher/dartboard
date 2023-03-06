terraform {
  required_version = "1.3.7"
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
  source                   = "./k3d_k3s"
  project_name             = local.project_name
  name                     = "upstream"
  network_name             = module.network.name
  server_count             = local.upstream_server_count
  agent_count              = local.upstream_agent_count
  distro_version           = local.upstream_distro_version
  sans                     = [local.upstream_san]
  kubernetes_api_port      = local.upstream_kubernetes_api_port
  additional_port_mappings = [[8443, 443]]
}

provider "helm" {
  kubernetes {
    config_path    = "~/.kube/config"
    config_context = "k3d-${local.project_name}-upstream"
  }
}

module "rancher" {
  depends_on   = [module.upstream_cluster]
  count        = local.upstream_server_count > 0 ? 1 : 0
  source       = "./rancher"
  public_name  = local.upstream_san
  private_name = module.upstream_cluster.first_server_private_name
  chart        = local.rancher_chart
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
