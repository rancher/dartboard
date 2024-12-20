terraform {
  required_providers {
    ssh = {
      source = "loafoe/ssh"
    }
  }
}

module "server_nodes" {
  count                 = var.server_count
  source                = "../node"
  project_name          = var.project_name
  name                  = "${var.name}-node-${count.index}"
  ssh_private_key_path  = var.ssh_private_key_path
  ssh_user              = var.ssh_user
  ssh_tunnels           = count.index == 0 ? var.additional_ssh_tunnels : []
  node_module           = var.node_module
  node_module_variables = var.node_module_variables
  network_config        = var.network_config
}

resource "ssh_sensitive_resource" "node_installation" {
  count        = var.server_count
  host         = module.server_nodes[count.index].private_name
  private_key  = file(var.ssh_private_key_path)
  user         = var.ssh_user
  bastion_host = var.network_config.ssh_bastion_host
  bastion_user = var.network_config.ssh_user
  timeout      = "600s"

  file {
    content = templatefile("${path.module}/install_etcd.sh", {
      etcd_version = var.etcd_version,
      etcd_token   = "${var.project_name}-${var.name}-token"
      etcd_name    = "${var.project_name}-${var.name}-${count.index}"
      server_name  = module.server_nodes[count.index].private_name
      server_ip    = module.server_nodes[count.index].private_ip
      etcd_names   = formatlist("%s-%s", "${var.project_name}-${var.name}", range(0, var.server_count))
      server_names = [for node in module.server_nodes : node.private_name]
    })
    destination = "/root/install_etcd.sh"
    permissions = "0700"
  }

  commands = [
    "/root/install_etcd.sh > >(tee install_etcd.log) 2> >(tee install_etcd.err >&2)",
  ]
}
