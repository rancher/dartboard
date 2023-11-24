terraform {
  required_version = "1.5.6"
  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "3.0.2"
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
    k3d = {
      source  = "pvotal-tech/k3d"
      version = "0.0.6"
    }
  }
}

provider "azurerm" {
  skip_provider_registration = true
  features {}
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
  ssh_private_key_path = var.ssh_private_key_path
  ssh_public_key_path  = var.ssh_public_key_path
}


module "cluster" {
  // HACK: we need to wait for the module.network to complete before moving on to workaround
  // terraform issue: https://github.com/hashicorp/terraform-provider-azurerm/issues/16928
  depends_on = [module.network]

  count        = length(local.clusters)
  source       = "../../modules/azure_k3s"
  project_name = local.project_name
  name         = local.clusters[count.index].name
  server_count = local.clusters[count.index].server_count
  agent_count  = local.clusters[count.index].agent_count
  agent_labels = local.clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true" }]
  ] : []
  agent_taints = local.clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
  ] : []
  distro_version = local.clusters[count.index].distro_version
  os_image       = local.clusters[count.index].os_image
  size           = local.clusters[count.index].size

  sans                      = [local.clusters[count.index].local_name]
  local_kubernetes_api_port = local.first_local_kubernetes_api_port + count.index
  local_http_port           = local.first_local_http_port + count.index
  local_https_port          = local.first_local_https_port + count.index
  resource_group_name       = azurerm_resource_group.rg.name
  location                  = azurerm_resource_group.rg.location
  ssh_public_key_path       = var.ssh_public_key_path
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_bastion_host          = module.network.bastion_public_name
  subnet_id                 = module.network.private_subnet_id
}
