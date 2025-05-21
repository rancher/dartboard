provider "harvester" {
  kubeconfig = var.kubeconfig
}

module "network" {
  source              = "../../modules/harvester/network"
  project_name        = var.project_name
  namespace           = var.namespace
  network_details     = var.network
  ssh_public_key_path = var.ssh_public_key_path
  ssh_bastion_host    = var.ssh_bastion_host
  ssh_bastion_user    = var.ssh_bastion_user
  ssh_bastion_key_path = var.ssh_bastion_key_path
}

resource "harvester_image" "created" {
  count = var.create_image ? 1 : 0
  name = "${var.project_name}-opensuse156"
  namespace = var.namespace
  display_name = "${var.project_name}-opensuse156"
  source_type = "download"
  url = "https://download.opensuse.org/repositories/Cloud:/Images:/Leap_15.6/images/openSUSE-Leap-15.6.x86_64-NoCloud.qcow2"
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
  node_module                      = "harvester/node"
  ssh_user                         = var.ssh_user
  ssh_private_key_path             = var.ssh_private_key_path
  network_config                   = module.network.config
  image_id                         = var.create_image ? harvester_image.created[0].id : null
  first_kubernetes_api_port        = var.first_kubernetes_api_port
  first_app_http_port              = var.first_app_http_port
  first_app_https_port             = var.first_app_https_port
}
