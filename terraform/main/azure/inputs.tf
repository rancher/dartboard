locals {
  project_name = "st"

  upstream_cluster = {
    name           = "upstream"
    server_count   = 3
    agent_count    = 2
    distro_version = "v1.26.9+k3s1"
    agent_labels = [
      [{ key : "monitoring", value : "true" }]
    ]
    agent_taints = [
      [{ key : "monitoring", value : "true", effect : "NoSchedule " }]
    ]

    local_name = "upstream.local.gd"
    // azure-specific
    size = "Standard_B4as_v2"
    os_image = {
      publisher = "suse"
      offer     = "opensuse-leap-15-5"
      sku       = "gen2"
      version   = "latest"
    }
  }

  downstream_clusters = [
    for i in range(2) :
    {
      name           = "downstream-${i}"
      server_count   = 3
      agent_count    = 2
      distro_version = "v1.26.9+k3s1"
      agent_labels   = []
      agent_taints   = []

      local_name = "downstream-${i}.local.gd"
      # public_ip = false
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
    name           = "tester"
    server_count   = 1
    agent_count    = 0
    distro_version = "v1.26.9+k3s1"
    agent_labels   = []
    agent_taints   = []

    local_name = "upstream.local.gd"
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

  first_local_kubernetes_api_port = 7445
  first_local_http_port           = 9080
  first_local_https_port          = 9443

  // azure-specific
  location = "West Europe"
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
