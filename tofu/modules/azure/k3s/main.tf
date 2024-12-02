module "server_nodes" {
  count                = var.server_count
  source               = "../host"
  os_image             = var.os_image
  os_disk_type         = var.os_disk_type
  os_disk_size         = var.os_disk_size
  os_ephemeral_disk    = var.os_ephemeral_disk
  location             = var.location
  resource_group_name  = var.resource_group_name
  size                 = var.size
  is_spot              = var.is_spot
  project_name         = var.project_name
  name                 = "${var.name}-server-${count.index}"
  ssh_public_key_path  = var.ssh_public_key_path
  ssh_private_key_path = var.ssh_private_key_path
  subnet_id            = var.subnet_id
  storage_account_uri  = var.storage_account_uri

  ssh_bastion_host = var.ssh_bastion_host
  ssh_tunnels = count.index == 0 ? [
    [var.local_kubernetes_api_port, 6443],
    [var.tunnel_app_http_port, 80],
    [var.tunnel_app_https_port, 443],
  ] : []
  host_configuration_commands = var.host_configuration_commands
}

module "agent_nodes" {
  count                       = var.agent_count
  source                      = "../host"
  os_image                    = var.os_image
  os_disk_type                = var.os_disk_type
  os_disk_size                = var.os_disk_size
  os_ephemeral_disk           = var.os_ephemeral_disk
  location                    = var.location
  resource_group_name         = var.resource_group_name
  size                        = var.size
  is_spot                     = var.is_spot
  project_name                = var.project_name
  name                        = "${var.name}-agent-${count.index}"
  ssh_public_key_path         = var.ssh_public_key_path
  ssh_private_key_path        = var.ssh_private_key_path
  subnet_id                   = var.subnet_id
  storage_account_uri         = var.storage_account_uri
  ssh_bastion_host            = var.ssh_bastion_host
  host_configuration_commands = var.host_configuration_commands
}


module "k3s" {
  source       = "../../generic/k3s"
  project      = var.project_name
  name         = var.name
  server_names = [for node in module.server_nodes : node.private_name]
  agent_names  = [for node in module.agent_nodes : node.private_name]
  agent_labels = var.agent_labels
  agent_taints = var.agent_taints
  sans         = compact(concat(var.sans, var.server_count > 0 ? [module.server_nodes[0].private_name] : []))

  ssh_user                  = var.ssh_user
  ssh_private_key_path      = var.ssh_private_key_path
  ssh_bastion_host          = var.ssh_bastion_host
  local_kubernetes_api_port = var.local_kubernetes_api_port

  distro_version      = var.distro_version
  max_pods            = var.max_pods
  node_cidr_mask_size = var.node_cidr_mask_size
  datastore_endpoint  = null
}
