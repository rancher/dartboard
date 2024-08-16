module "server_nodes" {
  count                 = var.server_count
  source                = "../aws_host"
  ami                   = var.ami
  instance_type         = var.instance_type
  availability_zone     = var.availability_zone
  project_name          = var.project_name
  name                  = "${var.name}-server-${count.index}"
  ssh_key_name          = var.ssh_key_name
  ssh_private_key_path  = var.ssh_private_key_path
  ssh_user              = var.ssh_user
  ssh_bastion_host      = var.ssh_bastion_host
  ssh_bastion_user      = var.ssh_bastion_user
  subnet_id             = var.subnet_id
  vpc_security_group_id = var.vpc_security_group_id
  ssh_tunnels = count.index == 0 ? [
    [var.local_kubernetes_api_port, 6443],
    [var.tunnel_app_http_port, 80],
    [var.tunnel_app_https_port, 443],
  ] : []
  host_configuration_commands = var.host_configuration_commands
}

module "agent_nodes" {
  count                       = var.agent_count
  source                      = "../aws_host"
  ami                         = var.ami
  instance_type               = var.instance_type
  availability_zone           = var.availability_zone
  project_name                = var.project_name
  name                        = "${var.name}-agent-${count.index}"
  ssh_key_name                = var.ssh_key_name
  ssh_private_key_path        = var.ssh_private_key_path
  ssh_user                    = var.ssh_user
  ssh_bastion_host            = var.ssh_bastion_host
  ssh_bastion_user            = var.ssh_bastion_user
  subnet_id                   = var.subnet_id
  vpc_security_group_id       = var.vpc_security_group_id
  host_configuration_commands = var.host_configuration_commands
}

module "rke2" {
  source       = "../rke2"
  project      = var.project_name
  name         = var.name
  server_names = [for node in module.server_nodes : node.private_name]
  agent_names  = [for node in module.agent_nodes : node.private_name]
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
