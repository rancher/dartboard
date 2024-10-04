provider "harvester" {
  kubeconfig = var.kubeconfig
}

module "network" {
  source              = "../../modules/harvester_network"
  create              = var.networks[0].create
  network_name        = var.networks[0].name
  clusternetwork_name = var.networks[0].clusternetwork_name
  namespace           = var.namespace
  ssh_public_key_path = var.ssh_public_key_path
}

module "images" {
  source       = "../../modules/harvester_images"
  project_name = var.project_name
  namespace    = var.namespace
  create       = anytrue([for c in local.k3s_clusters : c.image_name == null]) || anytrue([for c in local.rke2_clusters : c.image_name == null])
}

module "k3s_cluster" {
  count  = length(local.k3s_clusters)
  source = "../../modules/harvester_k3s"
  providers = {
    harvester = harvester
  }
  project_name     = var.project_name
  name             = local.k3s_clusters[count.index].name_prefix
  namespace        = var.namespace
  default_image_id = module.images.opensuse156_id
  image_name       = local.k3s_clusters[count.index].image_name
  image_namespace  = coalesce(local.k3s_clusters[count.index].image_namespace, var.namespace)
  tags             = local.k3s_clusters[count.index].tags
  cpu              = local.k3s_clusters[count.index].cpu
  memory           = local.k3s_clusters[count.index].memory
  disks            = local.k3s_clusters[count.index].disks
  server_count     = local.k3s_clusters[count.index].server_count
  agent_count      = local.k3s_clusters[count.index].agent_count
  agent_labels = local.k3s_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true" }]
  ] : []
  agent_taints = local.k3s_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
  ] : []
  distro_version              = local.k3s_clusters[count.index].distro_version
  sans                        = ["${local.k3s_clusters[count.index].name_prefix}.local.gd"]
  local_kubernetes_api_port   = var.first_kubernetes_api_port + count.index
  tunnel_app_http_port        = var.first_app_http_port + count.index
  tunnel_app_https_port       = var.first_app_https_port + count.index
  ssh_public_key              = module.network.ssh_public_key
  ssh_public_key_id           = module.network.ssh_public_key_id
  ssh_private_key_path        = var.ssh_private_key_path
  ssh_user                    = var.ssh_user
  password                    = var.password
  ssh_bastion_host            = var.ssh_bastion_host
  ssh_bastion_user            = var.ssh_bastion_user
  ssh_bastion_key_path        = var.ssh_private_key_path
  ssh_shared_public_keys      = var.ssh_shared_public_keys
  networks                    = var.networks
}

module "rke2_cluster" {
  count            = length(local.rke2_clusters)
  source           = "../../modules/harvester_rke2"
  project_name     = var.project_name
  name             = local.rke2_clusters[count.index].name_prefix
  namespace        = var.namespace
  default_image_id = module.images.opensuse156_id
  image_name       = local.rke2_clusters[count.index].image_name
  image_namespace  = coalesce(local.rke2_clusters[count.index].image_namespace, var.namespace)
  tags             = local.rke2_clusters[count.index].tags
  cpu              = local.rke2_clusters[count.index].cpu
  memory           = local.rke2_clusters[count.index].memory
  disks            = local.rke2_clusters[count.index].disks
  server_count     = local.rke2_clusters[count.index].server_count
  agent_count      = local.rke2_clusters[count.index].agent_count
  agent_labels = local.rke2_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true" }]
  ] : []
  agent_taints = local.rke2_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
  ] : []
  distro_version              = local.rke2_clusters[count.index].distro_version
  sans                        = ["${local.rke2_clusters[count.index].name_prefix}.local.gd"]
  local_kubernetes_api_port   = var.first_kubernetes_api_port + length(local.k3s_clusters) + count.index
  tunnel_app_http_port        = var.first_app_http_port + length(local.k3s_clusters) + count.index
  tunnel_app_https_port       = var.first_app_https_port + length(local.k3s_clusters) + count.index
  ssh_public_key              = module.network.ssh_public_key
  ssh_public_key_id           = module.network.ssh_public_key_id
  ssh_private_key_path        = var.ssh_private_key_path
  ssh_user                    = var.ssh_user
  password                    = var.password
  ssh_bastion_host            = var.ssh_bastion_host
  ssh_bastion_user            = var.ssh_bastion_user
  ssh_bastion_key_path        = var.ssh_private_key_path
  ssh_shared_public_keys      = var.ssh_shared_public_keys
  networks                    = var.networks
}
