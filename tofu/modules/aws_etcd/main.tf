module "server_nodes" {
  count                 = var.server_count
  source                = "../aws_host"
  ami                   = var.ami
  instance_type         = var.instance_type
  availability_zone     = var.availability_zone
  project_name          = var.project_name
  name                  = "${var.name}-node-${count.index}"
  ssh_key_name          = var.ssh_key_name
  ssh_private_key_path  = var.ssh_private_key_path
  ssh_user              = var.ssh_user
  subnet_id             = var.subnet_id
  vpc_security_group_id = var.vpc_security_group_id
  ssh_bastion_host      = var.ssh_bastion_host
  ssh_bastion_user      = var.ssh_bastion_user
  ssh_tunnels           = count.index == 0 ? var.additional_ssh_tunnels : []
}

module "etcd" {
  source       = "../etcd"
  project      = var.project_name
  name         = var.name
  server_names = [for node in module.server_nodes : node.private_name]
  server_ips   = [for node in module.server_nodes : node.private_ip]

  ssh_private_key_path = var.ssh_private_key_path
  ssh_bastion_host     = var.ssh_bastion_host

  etcd_version = var.etcd_version
}
