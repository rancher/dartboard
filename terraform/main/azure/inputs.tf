locals {
  project_name = "st"

  upstream_cluster = {
    name                        = "upstream"
    server_count                = 3
    agent_count                 = 2
    distro_version              = "1.26.6"
    reserve_node_for_monitoring = true

    // azure-specific
    size = "Standard_E2ads_v5"
    os_image = {
      publisher = "suse"
      offer     = "opensuse-leap-15-5"
      sku       = "gen2"
      version   = "latest"
    }
    os_disk_type = "StandardSSD_LRS"
    os_disk_size = 30
  }

  downstream_clusters = [
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
  }

  clusters = concat([local.upstream_cluster], local.downstream_clusters, [local.tester_cluster])

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
