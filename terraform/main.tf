terraform {
  required_version = "1.3.6"
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

module "postgres_kine" {
  depends_on            = [module.aws_network]
  source                = "./aws_postgres_kine"
  ami                   = local.postgres_ami
  availability_zone     = local.availability_zone
  project_name          = local.project_name
  name                  = "postgres"
  ssh_key_name          = module.aws_shared.key_name
  ssh_private_key_path  = local.ssh_private_key_path
  ssh_bastion_host      = module.bastion.public_name
  subnet_id             = module.aws_network.private_subnet_id
  vpc_security_group_id = module.aws_network.public_security_group_id
}

module "upstream_cluster" {
  source                        = "./aws_k3s"
  ami                           = local.upstream_ami
  instance_type                 = local.upstream_instance_type
  availability_zone             = local.availability_zone
  project_name                  = local.project_name
  name                          = "upstream"
  server_count                  = local.upstream_server_count
  agent_count                   = local.upstream_agent_count
  ssh_key_name                  = module.aws_shared.key_name
  ssh_private_key_path          = local.ssh_private_key_path
  ssh_bastion_host              = module.bastion.public_name
  subnet_id                     = module.aws_network.private_subnet_id
  vpc_security_group_id         = module.aws_network.private_security_group_id
  k8s_api_ssh_tunnel_local_port = 6443
  additional_ssh_tunnels        = [[3000, 443]]
  distro_version                = local.upstream_distro_version
  max_pods                      = local.upstream_max_pods
  node_cidr_mask_size           = local.upstream_node_cidr_mask_size
  sans                          = [local.upstream_san]
  secondary_subnet_id           = module.aws_network.secondary_private_subnet_id
  datastore_endpoint            = "http://${module.postgres_kine.private_name}:2379"
  host_configuration_commands   = ["apt update", "apt install -y python3-kubernetes python3-requests"]
}
