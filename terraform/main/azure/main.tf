terraform {
  required_version = "1.5.6"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.82.0"
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

locals {
  k3s_clusters  = [for cluster in local.clusters : cluster if strcontains(cluster.distro_version, "k3s")]
  rke2_clusters = [for cluster in local.clusters : cluster if strcontains(cluster.distro_version, "rke2")]
  aks_clusters  = [for cluster in local.clusters : cluster if !strcontains(cluster.distro_version, "v")]
}

provider "azurerm" {
  skip_provider_registration = true
  features {
    resource_group {
      # destroy resource group even if they contain stale resources
      prevent_deletion_if_contains_resources = false
    }
  }
}

resource "azurerm_resource_group" "rg" {
  name     = "${local.project_name}-rg"
  location = local.location
  tags     = local.tags
}

module "network" {
  source               = "../../modules/azure_network"
  project_name         = local.project_name
  location             = local.location
  resource_group_name  = azurerm_resource_group.rg.name
  ssh_public_key_path  = var.ssh_public_key_path
  ssh_private_key_path = var.ssh_private_key_path
}


module "k3s_cluster" {
  // HACK: we need to wait for the module.network to complete before moving on to workaround
  // terraform issue: https://github.com/hashicorp/terraform-provider-azurerm/issues/16928
  depends_on = [module.network]

  count        = length(local.k3s_clusters)
  source       = "../../modules/azure_k3s"
  project_name = local.project_name
  name         = local.k3s_clusters[count.index].name
  server_count = local.k3s_clusters[count.index].server_count
  agent_count  = local.k3s_clusters[count.index].agent_count
  agent_labels = local.k3s_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true" }]
  ] : []
  agent_taints = local.k3s_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
  ] : []
  distro_version = local.k3s_clusters[count.index].distro_version

  sans                      = ["${local.k3s_clusters[count.index].name}.local.gd"]
  local_kubernetes_api_port = local.first_local_kubernetes_api_port + count.index
  tunnel_app_http_port      = local.first_tunnel_app_http_port + count.index
  tunnel_app_https_port     = local.first_tunnel_app_https_port + count.index
  os_image                  = local.k3s_clusters[count.index].os_image
  size                      = local.k3s_clusters[count.index].size
  is_spot                   = lookup(local.k3s_clusters[count.index], "is_spot", false)
  os_disk_type              = lookup(local.k3s_clusters[count.index], "os_disk_type", "Standard_LRS")
  os_disk_size              = lookup(local.k3s_clusters[count.index], "os_disk_size", 30)
  resource_group_name       = azurerm_resource_group.rg.name
  location                  = azurerm_resource_group.rg.location
  ssh_public_key_path       = var.ssh_public_key_path
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_bastion_host          = module.network.bastion_public_name
  subnet_id                 = module.network.private_subnet_id
}

module "rke2_cluster" {
  // HACK: we need to wait for the module.network to complete before moving on to workaround
  // terraform issue: https://github.com/hashicorp/terraform-provider-azurerm/issues/16928
  depends_on = [module.network]

  count        = length(local.rke2_clusters)
  source       = "../../modules/azure_rke2"
  project_name = local.project_name
  name         = local.rke2_clusters[count.index].name
  server_count = local.rke2_clusters[count.index].server_count
  agent_count  = local.rke2_clusters[count.index].agent_count
  agent_labels = local.rke2_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true" }]
  ] : []
  agent_taints = local.rke2_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
  ] : []
  distro_version = local.rke2_clusters[count.index].distro_version

  sans                      = ["${local.rke2_clusters[count.index].name}.local.gd"]
  local_kubernetes_api_port = local.first_local_kubernetes_api_port + length(local.k3s_clusters) + count.index
  tunnel_app_http_port      = local.first_tunnel_app_http_port + length(local.k3s_clusters) + count.index
  tunnel_app_https_port     = local.first_tunnel_app_https_port + length(local.k3s_clusters) + count.index
  os_image                  = local.rke2_clusters[count.index].os_image
  size                      = local.rke2_clusters[count.index].size
  is_spot                   = lookup(local.rke2_clusters[count.index], "is_spot", false)
  os_disk_type              = lookup(local.rke2_clusters[count.index], "os_disk_type", "Standard_LRS")
  os_disk_size              = lookup(local.rke2_clusters[count.index], "os_disk_size", 30)
  resource_group_name       = azurerm_resource_group.rg.name
  location                  = azurerm_resource_group.rg.location
  ssh_public_key_path       = var.ssh_public_key_path
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_bastion_host          = module.network.bastion_public_name
  subnet_id                 = module.network.private_subnet_id
}

module "aks_cluster" {
  // HACK: we need to wait for the module.network to complete before moving on to workaround
  // terraform issue: https://github.com/hashicorp/terraform-provider-azurerm/issues/16928
  depends_on                 = [module.network]
  count                      = length(local.aks_clusters)
  source                     = "../../modules/azure_aks"
  project_name               = local.project_name
  name                       = local.aks_clusters[count.index].name
  system_node_pool_count     = local.aks_clusters[count.index].server_count
  main_node_pool_count       = local.aks_clusters[count.index].agent_count
  secondary_node_pool_count  = local.aks_clusters[count.index].reserve_node_for_monitoring ? 1 : 0
  secondary_node_pool_labels = local.aks_clusters[count.index].reserve_node_for_monitoring ? { monitoring : "true" } : {}
  secondary_node_pool_taints = local.aks_clusters[count.index].reserve_node_for_monitoring ? ["monitoring=true:NoSchedule"] : []
  distro_version             = local.aks_clusters[count.index].distro_version
  os_image                   = local.aks_clusters[count.index].os_image
  vm_size                    = local.aks_clusters[count.index].size

  resource_group_name = azurerm_resource_group.rg.name
  location            = azurerm_resource_group.rg.location
  subnet_id           = module.network.private_subnet_id
}
