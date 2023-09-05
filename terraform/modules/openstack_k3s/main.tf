module "server_nodes" {
  count                       = var.server_count
  source                      = "../openstack_host"
  availability_zone           = var.availability_zone
  flavor                      = var.flavor_name
  image                       = var.image_id
  project_name                = var.project_name
  name                        = "${var.name}-server-${count.index}"
  attach_floating_ip_from     = count.index == 0 ? var.floating_ip_pool_ext : null
  keypair                     = var.keypair
  ssh_private_key_path        = var.ssh_private_key_path
  network_id                  = var.network_id
  subnet_id                   = var.subnet_id
  ssh_bastion_host            = var.ssh_bastion_host
  host_configuration_commands = var.host_configuration_commands
}

module "agent_nodes" {
  count                       = var.agent_count
  source                      = "../openstack_host"
  availability_zone           = var.availability_zone
  image                       = var.image_id
  flavor                      = var.flavor_name
  project_name                = var.project_name
  name                        = "${var.name}-agent-${count.index}"
  keypair                     = var.keypair
  ssh_private_key_path        = var.ssh_private_key_path
  network_id                  = var.network_id
  subnet_id                   = var.subnet_id
  ssh_bastion_host            = var.ssh_bastion_host
  host_configuration_commands = var.host_configuration_commands
}


module "k3s" {
  source       = "../k3s"
  project      = var.project_name
  name         = var.name
  server_names = [for node in module.server_nodes : node.private_name]
  agent_names  = [for node in module.agent_nodes : node.private_name]
  agent_labels = var.agent_labels
  agent_taints = var.agent_taints
  sans         = concat([module.server_nodes[0].public_name], (var.server_count > 0 ? [module.server_nodes[0].private_name] : []))

  ssh_private_key_path = var.ssh_private_key_path
  ssh_bastion_host     = var.ssh_bastion_host

  distro_version      = var.distro_version
  max_pods            = var.max_pods
  node_cidr_mask_size = var.node_cidr_mask_size
  datastore_endpoint  = null
}
