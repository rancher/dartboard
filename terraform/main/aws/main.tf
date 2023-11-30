terraform {
  required_version = "1.5.6"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "4.31.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "4.0.3"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "2.7.1"
    }
    ssh = {
      source  = "loafoe/ssh"
      version = "2.2.1"
    }
  }
}

locals {
  k3s_clusters  = [for cluster in local.clusters : cluster if strcontains(cluster.distro_version, "k3s")]
  rke_clusters  = [for cluster in local.clusters : cluster if strcontains(cluster.distro_version, "rke_")]
  rke2_clusters = [for cluster in local.clusters : cluster if strcontains(cluster.distro_version, "rke2")]
}

provider "aws" {
  region = local.region
}

module "network" {
  source               = "../../modules/aws_network"
  project_name         = local.project_name
  region               = local.region
  availability_zone    = local.availability_zone
  ssh_public_key_path  = var.ssh_public_key_path
  ssh_private_key_path = var.ssh_private_key_path
}


module "k3s_cluster" {
  count        = length(local.k3s_clusters)
  source       = "../../modules/aws_k3s"
  project_name = local.project_name
  name         = local.k3s_clusters[count.index].name
  server_count = local.k3s_clusters[count.index].server_count
  agent_count  = local.k3s_clusters[count.index].agent_count
  agent_labels = local.k3s_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true" }]
  ] : []
  agent_taints = local.k3s_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
  ] : []
  distro_version = local.k3s_clusters[count.index].distro_version

  sans                      = ["${local.k3s_clusters[count.index].name}.local.gd"]
  local_kubernetes_api_port = local.first_local_kubernetes_api_port + count.index
  tunnel_app_http_port      = local.first_tunnel_app_http_port + count.index
  tunnel_app_https_port     = local.first_tunnel_app_https_port + count.index
  ami                       = local.k3s_clusters[count.index].ami
  instance_type             = local.k3s_clusters[count.index].instance_type
  availability_zone         = local.availability_zone
  ssh_key_name              = module.network.key_name
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_bastion_host          = module.network.bastion_public_name
  subnet_id                 = local.k3s_clusters[count.index].public_ip ? module.network.public_subnet_id : module.network.private_subnet_id
  vpc_security_group_id     = local.k3s_clusters[count.index].public_ip ? module.network.public_security_group_id : module.network.private_security_group_id
}

module "rke_cluster" {
  count        = length(local.rke_clusters)
  source       = "../../modules/aws_rke"
  project_name = local.project_name
  name         = local.rke_clusters[count.index].name
  server_count = local.rke_clusters[count.index].server_count
  agent_count  = local.rke_clusters[count.index].agent_count
  agent_labels = local.rke_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true" }]
  ] : []
  agent_taints = local.rke_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
  ] : []
  distro_version = local.rke_clusters[count.index].distro_version

  sans                      = ["${local.rke_clusters[count.index].name}.local.gd"]
  local_kubernetes_api_port = local.first_local_kubernetes_api_port + length(local.k3s_clusters) + count.index
  tunnel_app_http_port      = local.first_tunnel_app_http_port + length(local.k3s_clusters) + count.index
  tunnel_app_https_port     = local.first_tunnel_app_https_port + length(local.k3s_clusters) + count.index
  ami                       = local.rke_clusters[count.index].ami
  instance_type             = local.rke_clusters[count.index].instance_type
  availability_zone         = local.availability_zone
  ssh_key_name              = module.network.key_name
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_bastion_host          = module.network.bastion_public_name
  subnet_id                 = local.rke_clusters[count.index].public_ip ? module.network.public_subnet_id : module.network.private_subnet_id
  vpc_security_group_id     = local.rke_clusters[count.index].public_ip ? module.network.public_security_group_id : module.network.private_security_group_id
}

module "rke2_cluster" {
  count        = length(local.rke2_clusters)
  source       = "../../modules/aws_rke2"
  project_name = local.project_name
  name         = local.rke2_clusters[count.index].name
  server_count = local.rke2_clusters[count.index].server_count
  agent_count  = local.rke2_clusters[count.index].agent_count
  agent_labels = local.rke2_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true" }]
  ] : []
  agent_taints = local.rke2_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
  ] : []
  distro_version = local.rke2_clusters[count.index].distro_version

  sans                      = ["${local.rke2_clusters[count.index].name}.local.gd"]
  local_kubernetes_api_port = local.first_local_kubernetes_api_port + length(local.k3s_clusters) + length(local.rke_clusters) + count.index
  tunnel_app_http_port      = local.first_tunnel_app_http_port + length(local.k3s_clusters) + length(local.rke_clusters) + count.index
  tunnel_app_https_port     = local.first_tunnel_app_https_port + length(local.k3s_clusters) + length(local.rke_clusters) + count.index
  ami                       = local.rke2_clusters[count.index].ami
  instance_type             = local.rke2_clusters[count.index].instance_type
  availability_zone         = local.availability_zone
  ssh_key_name              = module.network.key_name
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_bastion_host          = module.network.bastion_public_name
  subnet_id                 = local.rke2_clusters[count.index].public_ip ? module.network.public_subnet_id : module.network.private_subnet_id
  vpc_security_group_id     = local.rke2_clusters[count.index].public_ip ? module.network.public_security_group_id : module.network.private_security_group_id
}
