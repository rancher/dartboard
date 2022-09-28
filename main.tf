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

module "bastion" {
  depends_on            = [module.aws_network]
  source                = "./aws_host"
  ami                   = local.bastion_ami
  instance_type         = local.bastion_instance_type
  availability_zone     = local.availability_zone
  project_name          = local.project_name
  name                  = "bastion"
  ssh_key_name          = module.aws_shared.key_name
  ssh_private_key_path  = local.ssh_private_key_path
  subnet_id             = module.aws_network.public_subnet_id
  vpc_security_group_id = module.aws_network.public_security_group_id
}

module "secrets" {
  source = "./secrets"
}

module "k3s" {
  depends_on           = [module.bastion]
  source               = "./k3s"
  project              = local.project_name
  name                 = module.bastion.public_name
  ssh_private_key_path = local.ssh_private_key_path
  k3s_version          = local.k3s_version

  client_ca_key          = module.secrets.client_ca_key
  client_ca_cert         = module.secrets.client_ca_cert
  server_ca_key          = module.secrets.server_ca_key
  server_ca_cert         = module.secrets.server_ca_cert
  request_header_ca_key  = module.secrets.request_header_ca_key
  request_header_ca_cert = module.secrets.request_header_ca_cert
}

provider "helm" {
  kubernetes {
    host                   = "https://${module.bastion.public_name}:6443"
    client_certificate     = module.secrets.master_user_cert
    client_key             = module.secrets.master_user_key
    cluster_ca_certificate = module.secrets.cluster_ca_certificate
  }
}

module "rancher" {
  depends_on       = [module.k3s]
  source           = "./rancher"
  public_name      = module.bastion.public_name
  private_name     = module.bastion.private_name
  api_token_string = module.secrets.api_token_string
  chart            = local.rancher_chart
}


// Downstream cluster


module "nodes" {
  depends_on            = [module.aws_network]
  quantity              = local.server_nodes + local.agent_nodes
  source                = "./aws_host"
  ami                   = local.nodes_ami
  instance_type         = local.nodes_instance_type
  availability_zone     = local.availability_zone
  project_name          = local.project_name
  name                  = "node"
  ssh_key_name          = module.aws_shared.key_name
  ssh_private_key_path  = local.ssh_private_key_path
  subnet_id             = module.aws_network.private_subnet_id
  vpc_security_group_id = module.aws_network.private_security_group_id
  ssh_bastion_host      = module.bastion.public_name
}

module "rke2" {
  depends_on   = [module.nodes]
  source       = "./rke2"
  project      = local.project_name
  server_names = slice(module.nodes.private_names, 0, local.server_nodes)
  agent_names  = slice(module.nodes.private_names, local.server_nodes, local.server_nodes + local.agent_nodes)

  ssh_private_key_path = local.ssh_private_key_path
  ssh_bastion_host     = module.bastion.public_name

  rke2_version = local.rke2_version
  max_pods     = local.max_pods

  client_ca_key          = module.secrets.client_ca_key
  client_ca_cert         = module.secrets.client_ca_cert
  server_ca_key          = module.secrets.server_ca_key
  server_ca_cert         = module.secrets.server_ca_cert
  request_header_ca_key  = module.secrets.request_header_ca_key
  request_header_ca_cert = module.secrets.request_header_ca_cert
}
