terraform {
  required_providers {
    ssh = {
      source  = "loafoe/ssh"
      version = "2.7.0"
    }
  }
}

locals {
  // Fall back to the node key when the network module does not define a
  // dedicated bastion key, so providers without one keep working.
  bastion_private_key = try(var.network_config.ssh_bastion_key_path, null) != null ? file(var.network_config.ssh_bastion_key_path) : file(var.ssh_private_key_path)
}

module "server_node" {
  source                = "../node"
  project_name          = var.project_name
  name                  = "${var.name}-server"
  ssh_private_key_path  = var.ssh_private_key_path
  ssh_user              = var.ssh_user
  node_module           = var.node_module
  node_module_variables = var.node_module_variables
  network_config        = var.network_config
}

resource "ssh_resource" "install_postgres" {
  depends_on = [module.server_node]

  host                = module.server_node.private_ip
  private_key         = file(var.ssh_private_key_path)
  user                = var.ssh_user
  bastion_host        = var.network_config.ssh_bastion_host
  bastion_user        = var.network_config.ssh_bastion_user
  bastion_private_key = local.bastion_private_key

  file {
    content = templatefile("${path.module}/install_postgres.sh", {
      kine_password = var.kine_password
    })
    destination = "/root/install_postgres.sh"
    permissions = "0700"
  }

  commands = [
    "/root/install_postgres.sh > >(tee install_postgres.log) 2> >(tee install_postgres.err >&2)"
  ]
}
