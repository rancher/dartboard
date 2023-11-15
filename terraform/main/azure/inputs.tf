locals {
  project_name = "fgiudici"

  upstream_cluster = {
    name           = "upstream"
    server_count   = 1
    agent_count    = 1
    distro_version = "v1.26.9+k3s1"
    agent_labels = [
      [{ key : "monitoring", value : "true" }]
    ]
    agent_taints = [
      [{ key : "monitoring", value : "true", effect : "NoSchedule " }]
    ]

    local_name = "upstream.local.gd"
    // azure-specific
    instance_type = "Standard_B2as_v2"
    os_image = {
      publisher = "suse"
      offer     = "opensuse-leap-15-5"
      sku       = "gen2"
      version   = "latest"
    }
  }

  clusters = concat([local.upstream_cluster])

  first_local_kubernetes_api_port = 7445
  first_local_http_port           = 9080
  first_local_https_port          = 9443

  // azure-specific
  location = "West Europe"
  tags = {
    Owner = "fgiudici"
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
