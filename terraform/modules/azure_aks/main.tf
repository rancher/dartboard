resource "azurerm_kubernetes_cluster" "cluster" {
  name                = "${var.project_name}-${var.name}"
  location            = var.location
  resource_group_name = var.resource_group_name
  kubernetes_version  = var.distro_version
  dns_prefix          = "${var.project_name}-${var.name}"
  sku_tier            = "Standard"

  web_app_routing {
    dns_zone_id = ""
  }

  network_profile {
    network_plugin    = "azure"
    load_balancer_sku = "standard"
  }

  default_node_pool {
    name                        = "default"
    temporary_name_for_rotation = "tempdefault"
    node_count                  = var.default_node_pool_count
    vm_size                     = var.vm_size
    vnet_subnet_id              = var.subnet_id
    os_disk_type                = var.os_ephemeral_disk ? "Ephemeral" : "Managed"
    os_disk_size_gb             = var.os_disk_size
    max_pods                    = 250
  }

  identity {
    type = "SystemAssigned"
  }

  dynamic "oms_agent" {
    for_each = var.enable_audit_log ? [1] : []
    content {
      log_analytics_workspace_id = azurerm_log_analytics_workspace.audit_log_workspace[0].id
    }
  }
}

resource "azurerm_kubernetes_cluster_node_pool" "secondary" {
  kubernetes_cluster_id = azurerm_kubernetes_cluster.cluster.id
  name                  = "secondary"
  vm_size               = var.secondary_vm_size
  node_count            = var.secondary_node_pool_count
  node_labels           = var.secondary_node_pool_labels
  node_taints           = var.secondary_node_pool_taints
  vnet_subnet_id        = var.subnet_id
  os_disk_type          = var.os_ephemeral_disk ? "Ephemeral" : "Managed"
  os_disk_size_gb       = var.os_disk_size
  max_pods              = 250
}

resource "local_file" "kubeconfig" {
  content = azurerm_kubernetes_cluster.cluster.kube_config_raw

  filename        = "${path.root}/config/${var.name}.yaml"
  file_permission = "0700"
}

resource "azurerm_log_analytics_workspace" "audit_log_workspace" {
  count               = var.enable_audit_log ? 1 : 0
  name                = "${var.project_name}-${var.name}-audit-log"
  location            = var.location
  resource_group_name = var.resource_group_name
}

resource "azurerm_monitor_diagnostic_setting" "audit_log_setting" {
  count                          = var.enable_audit_log ? 1 : 0
  name                           = "${var.project_name}-${var.name}-audit-log"
  target_resource_id             = azurerm_kubernetes_cluster.cluster.id
  log_analytics_workspace_id     = azurerm_log_analytics_workspace.audit_log_workspace[0].id
  log_analytics_destination_type = "Dedicated"

  enabled_log {
    # see https://learn.microsoft.com/en-us/azure/aks/monitor-aks-reference
    category = "kube-audit"
  }
}
