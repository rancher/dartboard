provider "aws" {
  region  = var.region
  profile = var.aws_profile
}

module "network" {
  source               = "../../modules/aws/network"
  project_name         = var.project_name
  region               = var.region
  availability_zone    = var.availability_zone
  existing_vpc_name    = var.existing_vpc_name
  bastion_host_ami     = length(var.bastion_host_ami) > 0 ? var.bastion_host_ami : null
  bastion_host_instance_type = "t4g.xlarge"
  ssh_bastion_user     = var.ssh_bastion_user
  ssh_public_key_path  = var.ssh_public_key_path
  ssh_private_key_path = var.ssh_private_key_path
  ssh_prefix_list      = var.ssh_prefix_list
}

module "test_environment" {
  source                           = "../../modules/generic/test_environment"
  project_name                     = var.project_name
  upstream_cluster                 = var.upstream_cluster
  upstream_cluster_distro_module   = var.upstream_cluster_distro_module
  downstream_cluster_templates     = var.downstream_cluster_templates
  downstream_cluster_distro_module = var.downstream_cluster_distro_module
  tester_cluster                   = var.tester_cluster
  tester_cluster_distro_module     = var.tester_cluster_distro_module
  node_module                      = "aws/node"
  ssh_user                         = var.ssh_user
  ssh_private_key_path             = var.ssh_private_key_path
  network_config                   = module.network.config
  first_kubernetes_api_port        = var.first_kubernetes_api_port
  first_app_http_port              = var.first_app_http_port
  first_app_https_port             = var.first_app_https_port
}
