module "network" {
  source       = "../../modules/k3d_network"
  project_name = var.project_name
}

module "cluster" {
  count          = length(local.all_clusters)
  source         = "../../modules/k3d_k3s"
  project_name   = var.project_name
  name           = local.all_clusters[count.index].name
  server_count   = local.all_clusters[count.index].server_count
  agent_count    = local.all_clusters[count.index].agent_count
  distro_version = local.all_clusters[count.index].distro_version

  sans                  = ["${local.all_clusters[count.index].name}.local.gd"]
  kubernetes_api_port   = var.first_kubernetes_api_port + count.index
  app_http_port         = var.first_app_http_port + count.index
  app_https_port        = var.first_app_https_port + count.index
  network_name          = module.network.name
  pull_proxy_registries = module.network.pull_proxy_registries
  enable_audit_log      = local.all_clusters[count.index].name == "upstream"
}
