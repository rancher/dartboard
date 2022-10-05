terraform {
  required_version = "1.2.9"
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
    tls = {
      source = "hashicorp/tls"
    }
    random = {
      source = "hashicorp/random"
    }
    helm = {
      source = "hashicorp/helm"
    }
    rancher2 = {
      source = "rancher/rancher2"
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
  source            = "./aws_network"
  region            = local.region
  availability_zone = local.availability_zone
  project_name      = local.project_name
}

module "secrets" {
  source = "./secrets"
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

module "upstream_server_nodes" {
  depends_on            = [module.aws_network]
  count                 = local.upstream_server_count
  source                = "./aws_host"
  ami                   = local.upstream_ami
  instance_type         = local.upstream_instance_type
  availability_zone     = local.availability_zone
  project_name          = local.project_name
  name                  = "upstream-server-node-${count.index}"
  ssh_key_name          = module.aws_shared.key_name
  ssh_private_key_path  = local.ssh_private_key_path
  subnet_id             = module.aws_network.private_subnet_id
  vpc_security_group_id = module.aws_network.private_security_group_id
  ssh_bastion_host      = module.bastion.public_name
  ssh_tunnels           = count.index == 0 ? [[local.upstream_local_port, 6443], [3000, 443]] : []
}

module "upstream_agent_nodes" {
  depends_on            = [module.aws_network]
  count                 = local.upstream_agent_count
  source                = "./aws_host"
  ami                   = local.upstream_ami
  instance_type         = local.upstream_instance_type
  availability_zone     = local.availability_zone
  project_name          = local.project_name
  name                  = "upstream-agent-node-${count.index}"
  ssh_key_name          = module.aws_shared.key_name
  ssh_private_key_path  = local.ssh_private_key_path
  subnet_id             = module.aws_network.private_subnet_id
  vpc_security_group_id = module.aws_network.private_security_group_id
  ssh_bastion_host      = module.bastion.public_name
}

module "upstream_rke2" {
  source       = "./rke2"
  project      = local.project_name
  name         = "upstream"
  server_names = [for node in module.upstream_server_nodes : node.private_name]
  agent_names  = [for node in module.upstream_agent_nodes : node.private_name]
  sans         = [local.upstream_san]

  ssh_private_key_path = local.ssh_private_key_path
  ssh_bastion_host     = module.bastion.public_name
  ssh_local_port       = local.upstream_local_port

  rke2_version        = local.upstream_rke2_version
  max_pods            = local.upstream_max_pods
  node_cidr_mask_size = local.upstream_node_cidr_mask_size

  client_ca_key          = module.secrets.client_ca_key
  client_ca_cert         = module.secrets.client_ca_cert
  server_ca_key          = module.secrets.server_ca_key
  server_ca_cert         = module.secrets.server_ca_cert
  request_header_ca_key  = module.secrets.request_header_ca_key
  request_header_ca_cert = module.secrets.request_header_ca_cert
  master_user_cert       = module.secrets.master_user_cert
  master_user_key        = module.secrets.master_user_key
}

provider "helm" {
  kubernetes {
    host                   = "https://${local.upstream_san}:6443"
    client_certificate     = module.secrets.master_user_cert
    client_key             = module.secrets.master_user_key
    cluster_ca_certificate = module.secrets.cluster_ca_certificate
  }
}

module "rancher" {
  depends_on       = [module.upstream_rke2, module.upstream_server_nodes]
  count            = local.upstream_server_count > 0 ? 1 : 0
  source           = "./rancher"
  public_name      = local.upstream_san
  private_name     = module.upstream_server_nodes[0].private_name
  api_token_string = module.secrets.api_token_string
  chart            = local.rancher_chart
}

module "downstream_server_nodes" {
  depends_on            = [module.aws_network]
  count                 = local.downstream_server_count
  source                = "./aws_host"
  ami                   = local.downstream_ami
  instance_type         = local.downstream_instance_type
  availability_zone     = local.availability_zone
  project_name          = local.project_name
  name                  = "downstream-server-node-${count.index}"
  ssh_key_name          = module.aws_shared.key_name
  ssh_private_key_path  = local.ssh_private_key_path
  subnet_id             = module.aws_network.private_subnet_id
  vpc_security_group_id = module.aws_network.private_security_group_id
  ssh_bastion_host      = module.bastion.public_name
  ssh_tunnels           = count.index == 0 ? [[local.downstream_local_port, 6443]] : []
}

module "downstream_agent_nodes" {
  depends_on            = [module.aws_network]
  count                 = local.downstream_agent_count
  source                = "./aws_host"
  ami                   = local.downstream_ami
  instance_type         = local.downstream_instance_type
  availability_zone     = local.availability_zone
  project_name          = local.project_name
  name                  = "downstream-agent-node-${count.index}"
  ssh_key_name          = module.aws_shared.key_name
  ssh_private_key_path  = local.ssh_private_key_path
  subnet_id             = module.aws_network.private_subnet_id
  vpc_security_group_id = module.aws_network.private_security_group_id
  ssh_bastion_host      = module.bastion.public_name
}

module "downstream_rke2" {
  source       = "./rke2"
  project      = local.project_name
  name         = "downstream"
  server_names = [for node in module.downstream_server_nodes : node.private_name]
  agent_names  = [for node in module.downstream_agent_nodes : node.private_name]
  sans         = [local.downstream_san]

  ssh_private_key_path = local.ssh_private_key_path
  ssh_bastion_host     = module.bastion.public_name
  ssh_local_port       = local.downstream_local_port

  rke2_version        = local.downstream_rke2_version
  max_pods            = local.downstream_max_pods
  node_cidr_mask_size = local.downstream_node_cidr_mask_size

  client_ca_key          = module.secrets.client_ca_key
  client_ca_cert         = module.secrets.client_ca_cert
  server_ca_key          = module.secrets.server_ca_key
  server_ca_cert         = module.secrets.server_ca_cert
  request_header_ca_key  = module.secrets.request_header_ca_key
  request_header_ca_cert = module.secrets.request_header_ca_cert
  master_user_cert       = module.secrets.master_user_cert
  master_user_key        = module.secrets.master_user_key
}
