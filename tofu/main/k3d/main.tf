provider "k3d" {
  fixes = {
      "dns" = false
  }
}

module "network" {
  source       = "../../modules/k3d/network"
  project_name = var.project_name
}

module "test_environment" {
  source                           = "../../modules/generic/test_environment"
  upstream_cluster                 = var.upstream_cluster
  upstream_cluster_distro_module   = var.upstream_cluster_distro_module
  downstream_cluster_templates     = var.downstream_cluster_templates
  downstream_cluster_distro_module = var.downstream_cluster_distro_module
  tester_cluster                   = var.tester_cluster
  tester_cluster_distro_module     = var.tester_cluster_distro_module
  node_module                      = null
  ssh_user                         = null
  ssh_private_key_path             = null
  network_config                   = module.network.config
  first_kubernetes_api_port        = var.first_kubernetes_api_port
  first_app_http_port              = var.first_app_http_port
  first_app_https_port             = var.first_app_https_port
}
