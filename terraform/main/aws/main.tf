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
    k3d = {
      source  = "pvotal-tech/k3d"
      version = "0.0.6"
    }
  }
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

module "cluster" {
  count          = length(local.clusters)
  source         = "../../modules/aws_k3s"
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
  ami                       = local.clusters[count.index].ami
  instance_type             = local.clusters[count.index].instance_type
  availability_zone         = local.availability_zone
  ssh_key_name              = module.network.key_name
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_bastion_host          = module.network.bastion_public_name
  subnet_id                 = module.network.private_subnet_id
  vpc_security_group_id     = module.network.private_security_group_id
}
