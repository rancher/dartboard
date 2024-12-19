module "network" {
  source       = "../../modules/k3d/network"
  project_name = var.project_name
}

module "test_environment" {
  source                           = "../../modules/generic/test_environment"
  upstream_cluster                 = var.upstream_cluster
  downstream_cluster_templates     = var.downstream_cluster_templates
  downstream_cluster_distro_module = var.downstream_cluster_distro_module
  tester_cluster                   = var.tester_cluster
  backend                          = "k3d"
  ssh_user                         = null
  ssh_private_key_path             = null
  network_backend_variables        = module.network.backend_variables
  first_kubernetes_api_port        = var.first_kubernetes_api_port
  first_app_http_port              = var.first_app_http_port
  first_app_https_port             = var.first_app_https_port
}
