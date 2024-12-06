provider "aws" {
  region  = var.region
  profile = var.aws_profile
}

module "network" {
  source               = "../../modules/aws/network"
  project_name         = var.project_name
  region               = var.region
  availability_zone    = var.availability_zone
  bastion_host_ami     = length(var.bastion_host_ami) > 0 ? var.bastion_host_ami : null
  ssh_user             = var.ssh_user
  ssh_public_key_path  = var.ssh_public_key_path
  ssh_private_key_path = var.ssh_private_key_path
}

module "test_environment" {
  source                       = "../../modules/generic/test_environment"
  upstream_cluster             = var.upstream_cluster
  downstream_cluster_templates = var.downstream_cluster_templates
  tester_cluster               = var.tester_cluster
  backend                      = "aws"
  network_backend_variables    = module.network.backend_variables
}
