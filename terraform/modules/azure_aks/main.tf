resource "azurerm_kubernetes_cluster" "cluster" {
  name                = "${var.project_name}-${var.name}"
  location            = var.location
  resource_group_name = var.resource_group_name
  kubernetes_version  = var.distro_version
  dns_prefix          = "${var.project_name}-${var.name}"

  web_app_routing {
    dns_zone_id = ""
  }

  default_node_pool {
    name       = "system"
    node_count = var.system_node_pool_count
    vm_size    = var.vm_size
    #    vnet_subnet_id = var.subnet_id
  }

  identity {
    type = "SystemAssigned"
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "main" {
  kubernetes_cluster_id = azurerm_kubernetes_cluster.cluster.id
  name                  = "main"
  node_count            = var.main_node_pool_count
  vm_size               = var.vm_size
  #  vnet_subnet_id        = var.subnet_id
}

resource "azurerm_kubernetes_cluster_node_pool" "secondary" {
  kubernetes_cluster_id = azurerm_kubernetes_cluster.cluster.id
  name                  = "secondary"
  vm_size               = var.vm_size
  node_count            = var.secondary_node_pool_count
  node_labels           = var.secondary_node_pool_labels
  node_taints           = var.secondary_node_pool_taints
  #  vnet_subnet_id        = var.subnet_id
}

resource "local_file" "kubeconfig" {
  content = azurerm_kubernetes_cluster.cluster.kube_config_raw

  filename        = "${path.root}/config/${var.name}.yaml"
  file_permission = "0700"
}
