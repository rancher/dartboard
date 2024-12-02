terraform {
  required_providers {
    ssh = {
      source = "loafoe/ssh"
    }
  }
}

resource "ssh_sensitive_resource" "node_installation" {
  count        = length(var.server_names)
  host         = var.server_names[count.index]
  private_key  = file(var.ssh_private_key_path)
  user         = "root"
  bastion_host = var.ssh_bastion_host
  timeout      = "600s"

  file {
    content = templatefile("${path.module}/install_etcd.sh", {
      etcd_version = var.etcd_version,
      etcd_token   = "${var.project}-${var.name}-token"
      etcd_name    = "${var.project}-${var.name}-${count.index}"
      server_name  = var.server_names[count.index]
      server_ip    = var.server_ips[count.index]
      etcd_names   = formatlist("%s-%s", "${var.project}-${var.name}", range(0, length(var.server_names)))
      server_names = var.server_names
    })
    destination = "/root/install_etcd.sh"
    permissions = "0700"
  }

  commands = [
    "/root/install_etcd.sh > >(tee install_etcd.log) 2> >(tee install_etcd.err >&2)",
  ]
}
