locals {
  downstream_clusters = flatten([
    for i, template in var.downstream_cluster_templates : [
      for j in range(template.cluster_count) : merge(template, { name = "downstream-${i}-${j}" })
  ] if template.cluster_count > 0])
}

module "upstream_cluster" {
  source                      = "../../${var.upstream_cluster_distro_module}"
  project_name                = var.project_name
  name                        = "upstream"
  server_count                = var.upstream_cluster.server_count
  agent_count                 = var.upstream_cluster.agent_count
  distro_version              = var.upstream_cluster.distro_version
  reserve_node_for_monitoring = var.upstream_cluster.reserve_node_for_monitoring
  enable_audit_log            = var.upstream_cluster.enable_audit_log

  sans                      = ["upstream.local.gd"]
  local_kubernetes_api_port = var.first_kubernetes_api_port
  tunnel_app_http_port      = var.first_app_http_port
  tunnel_app_https_port     = var.first_app_https_port
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_user                  = var.ssh_user
  node_module               = var.node_module
  network_config            = var.network_config
  image_id                  = var.image_id
  node_module_variables     = var.upstream_cluster.node_module_variables
}

module "tester_cluster" {
  count                       = var.tester_cluster != null ? 1 : 0
  source                      = "../../${var.tester_cluster_distro_module}"
  project_name                = var.project_name
  name                        = "tester"
  server_count                = var.tester_cluster.server_count
  agent_count                 = var.tester_cluster.agent_count
  distro_version              = var.tester_cluster.distro_version
  reserve_node_for_monitoring = var.tester_cluster.reserve_node_for_monitoring
  enable_audit_log            = var.tester_cluster.enable_audit_log

  sans                      = ["tester.local.gd"]
  local_kubernetes_api_port = var.first_kubernetes_api_port + 1
  tunnel_app_http_port      = var.first_app_http_port + 1
  tunnel_app_https_port     = var.first_app_https_port + 1
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_user                  = var.ssh_user
  node_module               = var.node_module
  network_config            = var.network_config
  image_id                  = var.image_id
  node_module_variables     = var.tester_cluster.node_module_variables
}


module "downstream_clusters" {
  count                       = length(local.downstream_clusters)
  source                      = "../../${var.downstream_cluster_distro_module}"
  project_name                = var.project_name
  name                        = local.downstream_clusters[count.index].name
  server_count                = local.downstream_clusters[count.index].server_count
  agent_count                 = local.downstream_clusters[count.index].agent_count
  distro_version              = local.downstream_clusters[count.index].distro_version
  reserve_node_for_monitoring = local.downstream_clusters[count.index].reserve_node_for_monitoring
  enable_audit_log            = local.downstream_clusters[count.index].enable_audit_log

  sans                      = ["${local.downstream_clusters[count.index].name}.local.gd"]
  local_kubernetes_api_port = var.first_kubernetes_api_port + 2 + count.index
  tunnel_app_http_port      = var.first_app_http_port + 2 + count.index
  tunnel_app_https_port     = var.first_app_https_port + 2 + count.index
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_user                  = var.ssh_user
  node_module               = var.node_module
  network_config            = var.network_config
  image_id                  = var.image_id
  node_module_variables     = local.downstream_clusters[count.index].node_module_variables
}
