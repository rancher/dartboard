provider "harvester" {
  kubeconfig = var.kubeconfig
}

module "network" {
  source              = "../../modules/harvester_network"
  project_name        = var.project_name
  namespace           = var.namespace
  ssh_public_key_path = var.ssh_public_key_path
}

# module "k3s_cluster" {
#   count        = length(local.k3s_clusters)
#   source       = "../../modules/harvester_k3s"
#   project_name = var.project_name
#   name         = local.k3s_clusters[count.index].name_prefix
#   server_count = local.k3s_clusters[count.index].server_count
#   agent_count  = local.k3s_clusters[count.index].agent_count
#   agent_labels = local.k3s_clusters[count.index].reserve_node_for_monitoring ? [
#     [{ key : "monitoring", value : "true" }]
#   ] : []
#   agent_taints = local.k3s_clusters[count.index].reserve_node_for_monitoring ? [
#     [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
#   ] : []
#   distro_version = local.k3s_clusters[count.index].distro_version

#   sans                      = ["${local.k3s_clusters[count.index].name_prefix}.local.gd"]
#   local_kubernetes_api_port = var.first_kubernetes_api_port + count.index
#   tunnel_app_http_port      = var.first_app_http_port + count.index
#   tunnel_app_https_port     = var.first_app_https_port + count.index
#   ami                       = local.k3s_clusters[count.index].ami
#   instance_type             = local.k3s_clusters[count.index].instance_type
#   availability_zone         = var.availability_zone
#   ssh_key_name              = module.network.key_name
#   ssh_private_key_path      = var.ssh_private_key_path
#   ssh_user                  = var.ssh_user
#   ssh_bastion_host          = module.network.bastion_public_name
#   ssh_bastion_user          = var.ssh_bastion_user
#   subnet_id                 = local.k3s_clusters[count.index].public_ip ? module.network.public_subnet_id : module.network.private_subnet_id
#   vpc_security_group_id     = local.k3s_clusters[count.index].public_ip ? module.network.public_security_group_id : module.network.private_security_group_id
# }

module "rke2_cluster" {
  count           = length(local.rke2_clusters)
  source          = "../../modules/harvester_rke2"
  providers = {
    harvester = harvester
  }
  project_name    = var.project_name
  name            = local.rke2_clusters[count.index].name_prefix
  namespace       = var.namespace
  image_name      = local.rke2_clusters[count.index].image_name
  image_namespace = local.rke2_clusters[count.index].image_namespace
  tags            = local.rke2_clusters[count.index].tags
  cpu             = local.rke2_clusters[count.index].cpu
  memory          = local.rke2_clusters[count.index].memory
  disks           = var.disks
  server_count    = local.rke2_clusters[count.index].server_count
  agent_count     = local.rke2_clusters[count.index].agent_count
  agent_labels = local.rke2_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true" }]
  ] : []
  agent_taints = local.rke2_clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
  ] : []
  distro_version              = local.rke2_clusters[count.index].distro_version
  cloudinit_secrets           = local.cloudinit_secrets
  efi                         = var.efi
  secure_boot                 = var.secure_boot
  host_configuration_commands = var.host_configuration_commands
  sans                        = ["${local.rke2_clusters[count.index].name_prefix}.local.gd"]
  local_kubernetes_api_port   = var.first_kubernetes_api_port + length(local.k3s_clusters) + count.index
  tunnel_app_http_port        = var.first_app_http_port + length(local.k3s_clusters) + count.index
  tunnel_app_https_port       = var.first_app_https_port + length(local.k3s_clusters) + count.index
  ssh_public_key              = module.network.ssh_public_key
  ssh_public_key_id           = module.network.ssh_public_key_id
  ssh_private_key_path        = var.ssh_private_key_path
  user                        = var.user
  password                    = var.password
  ssh_bastion_host            = var.ssh_bastion_host
  ssh_bastion_user            = var.ssh_bastion_user
  ssh_bastion_key_path        = var.ssh_private_key_path
  networks                    = var.networks
}
