locals {
  server_names = compact(concat(flatten([for node in module.server_nodes : [ for net in node.public_network_interfaces: net.ip_address ]])))
  agent_names = compact(concat(flatten([for node in module.agent_nodes : [ for net in node.public_network_interfaces: net.ip_address ]])))
}

module "server_nodes" {
  count                = var.server_count
  source               = "../harvester_host"
  project_name         = var.project_name
  name                 = "${var.name}-server-${count.index}"
  namespace            = var.namespace
  tags                 = var.tags
  image_name           = var.image_name
  image_namespace      = var.image_namespace
  cpu                  = var.cpu
  memory               = var.memory
  disks                = var.disks
  efi                  = var.efi
  secure_boot          = var.secure_boot
  ssh_keys             = var.ssh_keys
  ssh_private_key_path = var.ssh_private_key_path
  ssh_user             = var.ssh_user
  ssh_bastion_host     = var.ssh_bastion_host
  ssh_bastion_user     = var.ssh_bastion_user
  ssh_bastion_key_path = var.ssh_bastion_key_path
  networks             = var.networks
  cloudinit_secrets    = var.cloudinit_secrets
  ssh_tunnels = count.index == 0 ? [
    [var.local_kubernetes_api_port, 6443],
    [var.tunnel_app_http_port, 80],
    [var.tunnel_app_https_port, 443],
  ] : []
  host_configuration_commands = var.host_configuration_commands
}

module "agent_nodes" {
  count           = var.agent_count
  source          = "../harvester_host"
  project_name    = var.project_name
  name            = "${var.name}-agent-${count.index}"
  namespace       = var.namespace
  tags            = var.tags
  image_name      = var.image_name
  image_namespace = var.image_namespace
  cpu             = var.cpu
  memory          = var.memory
  disks           = var.disks
  efi                         = var.efi
  secure_boot                 = var.secure_boot
  ssh_keys                    = var.ssh_keys
  ssh_private_key_path        = var.ssh_private_key_path
  ssh_user                    = var.ssh_user
  ssh_bastion_host            = var.ssh_bastion_host
  ssh_bastion_user            = var.ssh_bastion_user
  ssh_bastion_key_path        = var.ssh_bastion_key_path
  networks                    = var.networks
  cloudinit_secrets           = var.cloudinit_secrets
  host_configuration_commands = var.host_configuration_commands
}

module "rke2" {
  source       = "../rke2"
  project      = var.project_name
  name         = var.name
  server_names = [for node in module.server_nodes : node.public_address]
  agent_names  = [for node in module.agent_nodes : node.public_address]
  agent_labels = var.agent_labels
  agent_taints = var.agent_taints
  sans         = var.sans

  ssh_private_key_path      = var.ssh_private_key_path
  ssh_user                  = var.ssh_user
  ssh_bastion_host          = var.ssh_bastion_host
  ssh_bastion_user          = var.ssh_bastion_user
  local_kubernetes_api_port = var.local_kubernetes_api_port

  distro_version      = var.distro_version
  max_pods            = var.max_pods
  node_cidr_mask_size = var.node_cidr_mask_size
}
