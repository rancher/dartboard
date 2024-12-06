module "k3s_cluster" {
  count        = length(local.k3s_clusters)
  source       = "../k3s"
  project_name = var.project_name
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
  local_kubernetes_api_port = var.first_kubernetes_api_port + count.index
  tunnel_app_http_port      = var.first_app_http_port + count.index
  tunnel_app_https_port     = var.first_app_https_port + count.index
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_user                  = var.ssh_user
  backend                   = var.backend
  network_backend_variables = var.network_backend_variables
  host_backend_variables    = local.k3s_clusters[count.index].backend_variables
}

module "rke2_cluster" {
  count        = length(local.rke2_clusters)
  source       = "../rke2"
  project_name = var.project_name
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
  local_kubernetes_api_port = var.first_kubernetes_api_port + length(local.k3s_clusters) + count.index
  tunnel_app_http_port      = var.first_app_http_port + length(local.k3s_clusters) + count.index
  tunnel_app_https_port     = var.first_app_https_port + length(local.k3s_clusters) + count.index
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_user                  = var.ssh_user

  backend                   = var.backend
  backend_network_variables = var.network_backend_variables
  host_backend_variables    = local.rke2_clusters[count.index].backend_variables
}
