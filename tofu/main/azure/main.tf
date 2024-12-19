provider "azurerm" {
  skip_provider_registration = true
  features {
    resource_group {
      # destroy resource group even if they contain stale resources
      prevent_deletion_if_contains_resources = false
    }
  }
}

module "network" {
  source               = "../../modules/azure/network"
  project_name         = var.project_name
  location             = var.location
  tags                 = var.tags
  bastion_os_image     = var.bastion_os_image
  ssh_bastion_user     = var.ssh_bastion_user
  ssh_public_key_path  = var.ssh_public_key_path
  ssh_private_key_path = var.ssh_private_key_path
}

module "test_environment" {
  // HACK: we need to wait for the module.network to complete before moving on to workaround
  // provider issue: https://github.com/hashicorp/terraform-provider-azurerm/issues/16928
  depends_on = [module.network]

  source                           = "../../modules/generic/test_environment"
  upstream_cluster                 = var.upstream_cluster
  upstream_cluster_distro_module   = var.upstream_cluster_distro_module
  downstream_cluster_templates     = var.downstream_cluster_templates
  downstream_cluster_distro_module = var.downstream_cluster_distro_module
  tester_cluster                   = var.tester_cluster
  tester_cluster_distro_module     = var.tester_cluster_distro_module
  node_module                      = "azure/node"
  ssh_user                         = var.ssh_user
  ssh_private_key_path             = var.ssh_private_key_path
  network_backend_variables        = module.network.backend_variables
  first_kubernetes_api_port        = var.first_kubernetes_api_port
  first_app_http_port              = var.first_app_http_port
  first_app_https_port             = var.first_app_https_port
}
