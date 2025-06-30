terraform {
  required_providers {
    ssh = {
      source = "loafoe/ssh"
    }
  }
}

module "server_nodes" {
  count                = var.server_count
  source               = "../node"
  project_name         = var.project_name
  name                 = "${var.name}-server-${count.index}"
  ssh_private_key_path = var.ssh_private_key_path
  ssh_user             = var.ssh_user
  ssh_tunnels = count.index == 0 ? [
    [var.local_kubernetes_api_port, 6443],
    [var.tunnel_app_http_port, 80],
    [var.tunnel_app_https_port, 443],
  ] : []
  node_module           = var.node_module
  node_module_variables = var.node_module_variables
  network_config        = var.network_config
}

module "agent_nodes" {
  count                 = var.agent_count
  source                = "../node"
  project_name          = var.project_name
  name                  = "${var.name}-agent-${count.index}"
  ssh_private_key_path  = var.ssh_private_key_path
  ssh_user              = var.ssh_user
  node_module           = var.node_module
  node_module_variables = var.node_module_variables
  network_config        = var.network_config
}

resource "ssh_sensitive_resource" "first_server_installation" {
  count        = var.server_count > 0 ? 1 : 0
  host         = module.server_nodes[0].private_name
  private_key  = file(var.ssh_private_key_path)
  user         = var.ssh_user
  bastion_host = var.network_config.ssh_bastion_host
  bastion_user = var.network_config.ssh_bastion_user
  timeout      = "600s"

  file {
    content = templatefile("${path.module}/install_k3s.sh", {
      distro_version = var.distro_version,
      sans           = concat([module.server_nodes[0].private_name], var.sans)
      exec           = "server"
      token          = null
      server_url     = null
      cluster_init   = var.server_count > 1
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
      datastore_endpoint     = var.datastore_endpoint
    })
    destination = "/tmp/install_k3s.sh"
    permissions = "0700"
  }

  file {
    content     = file("${path.module}/wait_for_k8s.sh")
    destination = "/tmp/wait_for_k8s.sh"
    permissions = "0700"
  }

  commands = [
    "sudo /tmp/install_k3s.sh > >(tee install_k3s.log) 2> >(tee install_k3s.err >&2)",
    "sudo /tmp/wait_for_k8s.sh",
    "sudo cat /var/lib/rancher/k3s/server/node-token",
  ]
}

resource "ssh_resource" "additional_server_installation" {
  depends_on = [ssh_sensitive_resource.first_server_installation]
  count      = max(0, var.server_count - 1)

  host         = module.server_nodes[count.index + 1].private_name
  private_key  = file(var.ssh_private_key_path)
  user         = var.ssh_user
  bastion_host = var.network_config.ssh_bastion_host
  bastion_user = var.network_config.ssh_bastion_user
  timeout      = "600s"

  file {
    content = templatefile("${path.module}/install_k3s.sh", {
      distro_version = var.distro_version,
      sans           = [module.server_nodes[count.index + 1].private_name]
      exec           = "server"
      token          = ssh_sensitive_resource.first_server_installation[0].result
      server_url     = "https://${module.server_nodes[0].private_name}:6443"
      cluster_init   = false
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
      datastore_endpoint     = var.datastore_endpoint
    })
    destination = "/tmp/install_k3s.sh"
    permissions = "0700"
  }

  commands = [
    "sudo /tmp/install_k3s.sh > >(tee install_k3s.log) 2> >(tee install_k3s.err >&2)"
  ]
}

resource "ssh_resource" "agent_installation" {
  depends_on = [ssh_sensitive_resource.first_server_installation]
  count      = var.agent_count

  host         = module.agent_nodes[count.index].private_name
  private_key  = file(var.ssh_private_key_path)
  user         = var.ssh_user
  bastion_host = var.network_config.ssh_bastion_host
  bastion_user = var.network_config.ssh_bastion_user
  timeout      = "600s"

  file {
    content = templatefile("${path.module}/install_k3s.sh", {
      distro_version = var.distro_version,
      sans           = [module.agent_nodes[count.index].private_name]
      exec           = "agent"
      token          = ssh_sensitive_resource.first_server_installation[0].result
      server_url     = "https://${module.server_nodes[0].private_name}:6443"
      cluster_init   = false
      labels = var.reserve_node_for_monitoring && count.index == 0 ? [
        { key : "monitoring", value : "true" }
      ] : []
      taints = var.reserve_node_for_monitoring && count.index == 0 ? [
        { key : "monitoring", value : "true", effect : "NoSchedule" }
      ] : []

      client_ca_key          = tls_private_key.client_ca_key.private_key_pem
      client_ca_cert         = tls_self_signed_cert.client_ca_cert.cert_pem
      server_ca_key          = tls_private_key.server_ca_key.private_key_pem
      server_ca_cert         = tls_self_signed_cert.server_ca_cert.cert_pem
      request_header_ca_key  = tls_private_key.request_header_ca_key.private_key_pem
      request_header_ca_cert = tls_self_signed_cert.request_header_ca_cert.cert_pem
      sleep_time             = 0
      max_pods               = var.max_pods
      node_cidr_mask_size    = var.node_cidr_mask_size
      datastore_endpoint     = var.datastore_endpoint
    })
    destination = "/tmp/install_k3s.sh"
    permissions = "0700"
  }

  commands = [
    "sudo /tmp/install_k3s.sh > >(tee install_k3s.log) 2> >(tee install_k3s.err >&2)",
  ]
}


locals {
  local_kubernetes_api_url = "https://${var.sans[0]}:${var.local_kubernetes_api_port}"
}

resource "local_file" "kubeconfig" {
  content = yamlencode({
    apiVersion = "v1"
    clusters = [
      {
        cluster = {
          certificate-authority-data = base64encode(tls_self_signed_cert.server_ca_cert.cert_pem)
          server                     = var.create_tunnels ? local.local_kubernetes_api_url : "https://${module.server_nodes[0].public_name}:6443"
        }
        name = var.name
      }
    ]
    contexts = [
      {
        context = {
          cluster = var.name
          user = "admin@${var.name}"
        }
        name = var.name
      }
    ]
    current-context = var.name
    kind            = "Config"
    preferences     = {}
    users = [
      {
        user = {
          client-certificate-data : base64encode(tls_locally_signed_cert.master_user.cert_pem)
          client-key-data : base64encode(tls_private_key.master_user.private_key_pem)
        }
        name = "admin@${var.name}"
      }
    ]
  })

  filename        = "${path.root}/${terraform.workspace}_config/${var.name}.yaml"
  file_permission = "0700"
}
