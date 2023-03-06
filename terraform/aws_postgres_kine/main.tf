terraform {
  required_providers {
    ssh = {
      source = "loafoe/ssh"
    }
  }
}

module "server_node" {
  source                = "../aws_host"
  ami                   = var.ami
  instance_type         = var.instance_type
  availability_zone     = var.availability_zone
  project_name          = var.project_name
  name                  = "${var.name}-server"
  ssh_key_name          = var.ssh_key_name
  ssh_private_key_path  = var.ssh_private_key_path
  subnet_id             = var.subnet_id
  vpc_security_group_id = var.vpc_security_group_id
  ssh_bastion_host      = var.ssh_bastion_host
}

resource "ssh_resource" "install_postgres" {
  depends_on = [module.server_node]

  host         = module.server_node.private_name
  private_key  = file(var.ssh_private_key_path)
  user         = "root"
  bastion_host = var.ssh_bastion_host

  file {
    source      = var.kine_executable != null ? var.kine_executable : "/dev/null"
    destination = "/tmp/kine"
    permissions = "0755"
  }

  file {
    content = templatefile("${path.module}/install_postgres.sh", {
      gogc         = tostring(var.gogc)
      kine_version = var.kine_version
    })
    destination = "/root/install_postgres.sh"
    permissions = "0700"
  }

  commands = [
    "/root/install_postgres.sh > >(tee install_postgres.log) 2> >(tee install_postgres.err >&2)"
  ]
}

// connect with
// PGPASSWORD=kinepassword psql -U kineuser -h localhost kine

output "private_name" {
  value = module.server_node.private_name
}
