terraform {
  required_providers {
    ssh = {
      source = "loafoe/ssh"
    }
  }
}

resource "ssh_sensitive_resource" "first_server_installation" {
  count        = length(var.server_names) > 0 ? 1 : 0
  host         = var.server_names[0]
  private_key  = file(var.ssh_private_key_path)
  user         = var.ssh_user
  bastion_host = var.ssh_bastion_host
  bastion_user = var.ssh_bastion_user
  timeout      = "600s"

  file {
    content = templatefile("${path.module}/install_rke2.sh", {
      distro_version = var.distro_version,
      sans           = concat([var.server_names[0]], var.sans)
      type           = "server"
      token          = null
      server_url     = null
      labels         = []
      taints         = []

      client_ca_key          = tls_private_key.client_ca_key.private_key_pem
      client_ca_cert         = tls_self_signed_cert.client_ca_cert.cert_pem
      server_ca_key          = tls_private_key.server_ca_key.private_key_pem
      server_ca_cert         = tls_self_signed_cert.server_ca_cert.cert_pem
      request_header_ca_key  = tls_private_key.request_header_ca_key.private_key_pem
      request_header_ca_cert = tls_self_signed_cert.request_header_ca_cert.cert_pem
      sleep_time             = 0
      max_pods               = var.max_pods
      node_cidr_mask_size    = var.node_cidr_mask_size
    })
    destination = "/tmp/install_rke2.sh"
    permissions = "0700"
  }

  file {
    content     = file("${path.module}/wait_for_k8s.sh")
    destination = "/tmp/wait_for_k8s.sh"
    permissions = "0700"
  }

  commands = [
    "sudo /tmp/install_rke2.sh > >(tee install_rke2.log) 2> >(tee install_rke2.err >&2)",
    "sudo /tmp/wait_for_k8s.sh",
    "sudo cat /var/lib/rancher/rke2/server/node-token",
  ]
}

resource "ssh_resource" "additional_server_installation" {
  depends_on = [ssh_sensitive_resource.first_server_installation]
  count      = length(var.server_names) > 0 ? length(var.server_names) - 1 : 0

  host         = var.server_names[count.index + 1]
  private_key  = file(var.ssh_private_key_path)
  user         = var.ssh_user
  bastion_host = var.ssh_bastion_host
  bastion_user = var.ssh_bastion_user
  timeout      = "600s"

  file {
    content = templatefile("${path.module}/install_rke2.sh", {
      distro_version = var.distro_version,
      sans           = [var.server_names[count.index + 1]]
      type           = "server"
      token          = ssh_sensitive_resource.first_server_installation[0].result
      server_url     = "https://${var.server_names[0]}:9345"
      labels         = []
      taints         = []

      client_ca_key          = tls_private_key.client_ca_key.private_key_pem
      client_ca_cert         = tls_self_signed_cert.client_ca_cert.cert_pem
      server_ca_key          = tls_private_key.server_ca_key.private_key_pem
      server_ca_cert         = tls_self_signed_cert.server_ca_cert.cert_pem
      request_header_ca_key  = tls_private_key.request_header_ca_key.private_key_pem
      request_header_ca_cert = tls_self_signed_cert.request_header_ca_cert.cert_pem
      sleep_time             = count.index * 60
      max_pods               = var.max_pods
      node_cidr_mask_size    = var.node_cidr_mask_size
    })
    destination = "/tmp/install_rke2.sh"
    permissions = "0700"
  }

  commands = [
    "sudo /tmp/install_rke2.sh > >(tee install_rke2.log) 2> >(tee install_rke2.err >&2)"
  ]
}

resource "ssh_resource" "agent_installation" {
  depends_on = [ssh_sensitive_resource.first_server_installation]
  count      = length(var.agent_names)

  host         = var.agent_names[count.index]
  private_key  = file(var.ssh_private_key_path)
  user         = var.ssh_user
  bastion_host = var.ssh_bastion_host
  bastion_user = var.ssh_bastion_user
  timeout      = "600s"

  file {
    content = templatefile("${path.module}/install_rke2.sh", {
      distro_version = var.distro_version,
      sans           = [var.agent_names[count.index]]
      type           = "agent"
      token          = ssh_sensitive_resource.first_server_installation[0].result
      server_url     = "https://${var.server_names[0]}:9345"
      labels         = length(var.agent_labels) > count.index ? var.agent_labels[count.index] : []
      taints         = length(var.agent_taints) > count.index ? var.agent_taints[count.index] : []

      client_ca_key          = tls_private_key.client_ca_key.private_key_pem
      client_ca_cert         = tls_self_signed_cert.client_ca_cert.cert_pem
      server_ca_key          = tls_private_key.server_ca_key.private_key_pem
      server_ca_cert         = tls_self_signed_cert.server_ca_cert.cert_pem
      request_header_ca_key  = tls_private_key.request_header_ca_key.private_key_pem
      request_header_ca_cert = tls_self_signed_cert.request_header_ca_cert.cert_pem
      sleep_time             = 0
      max_pods               = var.max_pods
      node_cidr_mask_size    = var.node_cidr_mask_size
    })
    destination = "/tmp/install_rke2.sh"
    permissions = "0700"
  }

  commands = [
    "sudo /tmp/install_rke2.sh > >(tee install_rke2.log) 2> >(tee install_rke2.err >&2)"
  ]
}
