locals {
  project_name = "st"

  upstream_cluster = {
    name                        = "upstream"
    agent_count                 = 4
    distro_version              = "1.27.7"
    reserve_node_for_monitoring = true

    // azure-specific
    size              = "Standard_D8ds_v4"
    os_ephemeral_disk = true
    enable_audit_log  = true
  }

  downstream_k3s_clusters = [
    for i in range(2) :
    {
      name                        = "downstream-${i}"
      server_count                = 1
      agent_count                 = 0
      distro_version              = "v1.26.9+k3s1"
      reserve_node_for_monitoring = false

      // azure-specific
      size = "Standard_B2as_v2"
      os_image = {
        publisher = "suse"
        offer     = "opensuse-leap-15-5"
        sku       = "gen2"
        version   = "latest"
      }
      os_ephemeral_disk = false
    }
  ]

  downstream_aks_clusters = [
    for i in range(length(local.downstream_k3s_clusters), length(local.downstream_k3s_clusters) + 0) :
    {
      name                        = "downstream-${i}"
      agent_count                 = 1
      distro_version              = "1.27.7"
      reserve_node_for_monitoring = false

      // azure-specific
      size              = "Standard_B2as_v2"
      os_ephemeral_disk = false
      enable_audit_log  = false
    }
  ]

  tester_cluster = {
    name                        = "tester"
    server_count                = 1
    agent_count                 = 0
    distro_version              = "v1.26.9+k3s1"
    reserve_node_for_monitoring = false

    // azure-specific
    size = "Standard_B2as_v2"
    os_image = {
      publisher = "suse"
      offer     = "opensuse-leap-15-5"
      sku       = "gen2"
      version   = "latest"
    }
    os_ephemeral_disk = false
  }

  clusters = concat([local.upstream_cluster], local.downstream_k3s_clusters, local.downstream_aks_clusters, [local.tester_cluster])

  // azure-specific
  first_local_kubernetes_api_port = 8445
  first_tunnel_app_http_port      = 10080
  first_tunnel_app_https_port     = 10443
  location                        = "West Europe"
  tags = {
    Owner = local.project_name
  }
}

// azure supports RSA ssh key pairs only
variable "ssh_public_key_path" {
  description = "Path to SSH public key file (can be generated with `ssh-keygen -f ~/.ssh/azure_rsa -t rsa -b 4096`)"
  default     = "~/.ssh/azure_rsa.pub"
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key file (can be generated with `ssh-keygen -f ~/.ssh/azure_rsa -t rsa -b 4096`)"
  default     = "~/.ssh/azure_rsa"
}
