terraform {
  required_providers {
    ssh = {
      source = "loafoe/ssh"
    }
  }
}

resource "ssh_resource" "installation" {
  host        = var.name
  private_key = file(var.ssh_private_key_path)
  user        = "root"
  bastion_host        = var.ssh_bastion_host
  timeout             = "120s"

  file {
    content = templatefile("${path.module}/install_k3s.sh", {
      k3s_version=var.k3s_version
      name=var.name
      client_ca_key = var.client_ca_key
      client_ca_cert = var.client_ca_cert
      server_ca_key = var.server_ca_key
      server_ca_cert = var.server_ca_cert
      request_header_ca_key = var.request_header_ca_key
      request_header_ca_cert = var.request_header_ca_cert
    })
    destination = "/tmp/install_k3s.sh"
    permissions = "0700"
  }

  commands = [
      "/tmp/install_k3s.sh",
    ]
}
