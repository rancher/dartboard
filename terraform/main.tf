terraform {
  required_version = "1.3.7"
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
      source  = "moio/k3d"
      version = "0.0.7"
    }
  }
}

provider "aws" {
  region = local.region
}

module "aws_shared" {
  source              = "./aws_shared"
  project_name        = local.project_name
  ssh_public_key_path = local.ssh_public_key_path
}

module "aws_network" {
  source                      = "./aws_network"
  region                      = local.region
  availability_zone           = local.availability_zone
  secondary_availability_zone = local.secondary_availability_zone
  project_name                = local.project_name
}

module "bastion" {
  depends_on            = [module.aws_network]
  source                = "./aws_host"
  ami                   = local.bastion_ami
  availability_zone     = local.availability_zone
  project_name          = local.project_name
  name                  = "bastion"
  ssh_key_name          = module.aws_shared.key_name
  ssh_private_key_path  = local.ssh_private_key_path
  subnet_id             = module.aws_network.public_subnet_id
  vpc_security_group_id = module.aws_network.public_security_group_id
}

module "upstream_cluster" {
  source = "./aws_rke"
  # alternatives:
  # source = "./aws_k3s"
  # source = "./aws_rke2"
  ami                    = local.upstream_ami
  instance_type          = local.upstream_instance_type
  availability_zone      = local.availability_zone
  project_name           = local.project_name
  name                   = "upstream"
  server_count           = local.upstream_server_count
  agent_count            = local.upstream_agent_count
  ssh_key_name           = module.aws_shared.key_name
  ssh_private_key_path   = local.ssh_private_key_path
  ssh_bastion_host       = module.bastion.public_name
  subnet_id              = module.aws_network.private_subnet_id
  vpc_security_group_id  = module.aws_network.private_security_group_id
  kubernetes_api_port    = local.upstream_kubernetes_api_port
  additional_ssh_tunnels = [[local.rancher_port, 443]]
  distro_version         = local.upstream_distro_version
  sans                   = [local.upstream_san]
  # k3s only
  # secondary_subnet_id    = module.aws_network.secondary_private_subnet_id
}

provider "helm" {
  kubernetes {
    config_path = "../config/upstream.yaml"
  }
}

module "rancher" {
  depends_on         = [module.upstream_cluster]
  count              = local.upstream_server_count > 0 ? 1 : 0
  source             = "./rancher"
  public_name        = local.upstream_san
  private_name       = module.upstream_cluster.first_server_private_name
  chart              = local.rancher_chart
}

module "downstream_cluster" {
  source = "./aws_k3s"
  # alternatives:
  # source = "./aws_rke"
  # source = "./aws_rke2"
  ami                   = local.downstream_ami
  instance_type         = local.downstream_instance_type
  availability_zone     = local.availability_zone
  project_name          = local.project_name
  name                  = "downstream"
  server_count          = local.downstream_server_count
  agent_count           = local.downstream_agent_count
  ssh_key_name          = module.aws_shared.key_name
  ssh_private_key_path  = local.ssh_private_key_path
  ssh_bastion_host      = module.bastion.public_name
  subnet_id             = module.aws_network.private_subnet_id
  vpc_security_group_id = module.aws_network.private_security_group_id
  kubernetes_api_port   = local.downstream_kubernetes_api_port
  distro_version        = local.downstream_distro_version
  sans                  = [local.downstream_san]
  secondary_subnet_id   = module.aws_network.secondary_private_subnet_id
}
