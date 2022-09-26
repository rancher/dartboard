terraform {
  required_providers {
    ssh = {
      source = "loafoe/ssh"
    }
  }
}

locals {
  fist_server_name = var.server_names[0]
  server_url = "https://${local.fist_server_name}:9345"
}

resource "ssh_sensitive_resource" "first_server_installation" {
  host        = local.fist_server_name
  private_key = file(var.ssh_private_key_path)
  user        = "root"
  bastion_host        = var.ssh_bastion_host
  timeout             = "600s"

  file {
    content = templatefile("${path.module}/install_rke2.sh", {
      rke2_version=var.rke2_version,
      sans=concat([local.fist_server_name], var.sans)
      type = "server"
      token = null
      server_url = null

      client_ca_key = var.client_ca_key
      client_ca_cert = var.client_ca_cert
      server_ca_key = var.server_ca_key
      server_ca_cert = var.server_ca_cert
      request_header_ca_key = var.request_header_ca_key
      request_header_ca_cert = var.request_header_ca_cert
      sleep_time = 0
      max_pods = var.max_pods
    })
    destination = "/root/install_rke2.sh"
    permissions = "0700"
  }

  commands = [
    "/root/install_rke2.sh > >(tee install_rke2.log) 2> >(tee install_rke2.err >&2)",
    "cat /var/lib/rancher/rke2/server/node-token",
  ]
}

resource "ssh_resource" "additional_server_installation" {
  depends_on = [ssh_sensitive_resource.first_server_installation]
  count = length(var.server_names) - 1

  host        = var.server_names[count.index + 1]
  private_key = file(var.ssh_private_key_path)
  user        = "root"
  bastion_host        = var.ssh_bastion_host
  timeout             = "600s"

  file {
    content = templatefile("${path.module}/install_rke2.sh", {
      rke2_version=var.rke2_version,
      sans=[var.server_names[count.index + 1]]
      type = "server"
      token = ssh_sensitive_resource.first_server_installation.result
      server_url = local.server_url

      client_ca_key = var.client_ca_key
      client_ca_cert = var.client_ca_cert
      server_ca_key = var.server_ca_key
      server_ca_cert = var.server_ca_cert
      request_header_ca_key = var.request_header_ca_key
      request_header_ca_cert = var.request_header_ca_cert
      sleep_time = count.index * 60
      max_pods = var.max_pods
    })
    destination = "/root/install_rke2.sh"
    permissions = "0700"
  }

  commands = [
    "/root/install_rke2.sh > >(tee install_rke2.log) 2> >(tee install_rke2.err >&2)"
  ]
}

resource "ssh_resource" "agent_installation" {
  depends_on = [ssh_sensitive_resource.first_server_installation]
  count = length(var.agent_names)

  host        = var.agent_names[count.index]
  private_key = file(var.ssh_private_key_path)
  user        = "root"
  bastion_host        = var.ssh_bastion_host
  timeout             = "600s"

  file {
    content = templatefile("${path.module}/install_rke2.sh", {
      rke2_version=var.rke2_version,
      sans=[var.agent_names[count.index]]
      type = "agent"
      token = ssh_sensitive_resource.first_server_installation.result
      server_url = local.server_url

      client_ca_key = var.client_ca_key
      client_ca_cert = var.client_ca_cert
      server_ca_key = var.server_ca_key
      server_ca_cert = var.server_ca_cert
      request_header_ca_key = var.request_header_ca_key
      request_header_ca_cert = var.request_header_ca_cert
      sleep_time = 0
      max_pods = var.max_pods
    })
    destination = "/root/install_rke2.sh"
    permissions = "0700"
  }

  commands = [
    "/root/install_rke2.sh > >(tee install_rke2.log) 2> >(tee install_rke2.err >&2)"
  ]
}
