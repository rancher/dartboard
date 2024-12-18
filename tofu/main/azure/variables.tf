variable "ssh_public_key_path" {
  description = "Path to SSH public key file (can be generated with `ssh-keygen -t rsa -f ~/.ssh/azure_rsa`)"
  default     = "~/.ssh/azure_rsa.pub"
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key file (can be generated with `ssh-keygen -t rsa -f ~/.ssh/azure_rsa`)"
  default     = "~/.ssh/azure_rsa"
}

variable "ssh_user" {
  description = "User name to use for the SSH connection to all nodes in all clusters"
  default     = "azureuser"
}

variable "ssh_bastion_user" {
  description = "User name for the SSH bastion host's OS"
  default     = "azureuser"
}

# Upstream cluster specifics
variable "upstream_cluster" {
  type = object({
    server_count   = number // Number of server nodes in the upstream cluster
    agent_count    = number // Number of agent nodes in the upstream cluster
    distro_module  = string // Path to the module to use for the upstream cluster
    distro_version = string // Version of the Kubernetes distro in the upstream cluster

    public_ip                   = bool // Whether the upstream cluster should have a public IP assigned
    reserve_node_for_monitoring = bool // Set a 'monitoring' label and taint on one node of the upstream cluster to reserve it for monitoring
    enable_audit_log            = bool // Enable audit log for the cluster

    backend_variables = any // Backend-specific variables
  })
  default = {
    server_count                = 1
    agent_count                 = 0
    distro_module               = "generic/k3s"
    distro_version              = "v1.26.9+k3s1"
    public_ip                   = true
    reserve_node_for_monitoring = false
    enable_audit_log            = false

    backend_variables = {
      os_image = {
        publisher = "suse"
        offer     = "opensuse-leap-15-5"
        sku       = "gen2"
        version   = "latest"
      }
      size              = "Standard_D8ds_v4"
      is_spot           = false
      os_disk_type      = "StandardSSD_LRS"
      os_disk_size      = 30
      os_ephemeral_disk = true
    }
  }
}

# Downstream cluster specifics
variable "downstream_cluster_templates" {
  type = list(object({
    cluster_count  = number // Number of downstream clusters that should be created using this configuration
    server_count   = number // Number of server nodes in the downstream cluster
    agent_count    = number // Number of agent nodes in the downstream cluster
    distro_version = string // Version of the Kubernetes distro in the downstream cluster

    public_ip                   = bool // Whether the downstream cluster should have a public IP assigned
    reserve_node_for_monitoring = bool // Set a 'monitoring' label and taint on one node of the downstream cluster to reserve it for monitoring
    enable_audit_log            = bool // Enable audit log for the cluster

    backend_variables = any // Backend-specific variables
  }))
  default = [{
    cluster_count               = 0 // defaults to 0 to keep in-line with previous behavior
    server_count                = 1
    agent_count                 = 0
    distro_version              = "v1.26.9+k3s1"
    public_ip                   = false
    reserve_node_for_monitoring = false
    enable_audit_log            = false

    backend_variables = {
      os_image = {
        publisher = "suse"
        offer     = "opensuse-leap-15-5"
        sku       = "gen2"
        version   = "latest"
      }
      size              = "Standard_B1ms"
      is_spot           = false
      os_disk_type      = "StandardSSD_LRS"
      os_disk_size      = 30
      os_ephemeral_disk = false
    }
  }]
}

# Note: this is kept constant for all templates because OpenTofu v1.8.2 does not allow to use
# each.value, each.key or count.index in expressions for module paths
# context is https://github.com/opentofu/opentofu/blob/main/rfc/20240513-static-evaluation/module-expansion.md ->
# https://github.com/opentofu/opentofu/issues/1896#issuecomment-2275763570 ->
# https://github.com/opentofu/opentofu/issues/2155
variable "downstream_cluster_distro_module" {
  description = "Name of the module to use for the downstream clusters"
  default     = "generic/k3s"
}

# Tester cluster specifics
variable "tester_cluster" {
  type = object({
    server_count   = number // Number of server nodes in the tester cluster
    agent_count    = number // Number of agent nodes in the tester cluster
    distro_module  = string // Path to the module to use for the tester cluster
    distro_version = string // Version of the Kubernetes distro in the tester cluster

    public_ip                   = bool // Whether the tester cluster should have a public IP assigned
    reserve_node_for_monitoring = bool // Set a 'monitoring' label and taint on one node of the tester cluster to reserve it for monitoring
    enable_audit_log            = bool // Enable audit log for the cluster

    backend_variables = any // Backend-specific variables
  })
  default = {
    server_count                = 1
    agent_count                 = 0
    distro_module               = "generic/k3s"
    distro_version              = "v1.26.9+k3s1"
    public_ip                   = false
    reserve_node_for_monitoring = false
    enable_audit_log            = false

    backend_variables = {
      size = "Standard_B2as_v2"
      os_image = {
        publisher = "suse"
        offer     = "opensuse-leap-15-5"
        sku       = "gen2"
        version   = "latest"
      }
      is_spot           = false
      os_disk_type      = "StandardSSD_LRS"
      os_disk_size      = 30
      os_ephemeral_disk = false
    }
  }
}

variable "deploy_tester_cluster" {
  description = "Use false not to deploy a tester cluster"
  default     = true
}

# "Multi-tenancy" variables
variable "project_name" {
  description = "Name of this project, used as prefix for resources it creates"
  default     = "st"
}

variable "first_kubernetes_api_port" {
  description = "Port number where the Kubernetes API of the first cluster is published locally. Other clusters' ports are published in successive ports"
  default     = 8445
}

variable "first_app_http_port" {
  description = "Port number where the first server's port 80 is published locally. Other clusters' ports are published in successive ports"
  default     = 10080
}

variable "first_app_https_port" {
  description = "Port number where the first server's port 443 is published locally. Other clusters' ports are published in successive ports"
  default     = 10443
}

# Backend-specific variables
variable "location" {
  description = "Azure Location where the instance in created"
  default     = "West Europe"
}

variable "tags" {
  description = "Tags to be applied to all resources created by this module"
  type        = map(string)
  default = {
    Owner = "st"
  }
}

variable "bastion_os_image" {
  description = "OS image to use for the bastion host"
  type = object({
    publisher = string
    offer     = string
    sku       = string
    version   = string
  })
  default = {
    publisher = "suse"
    offer     = "opensuse-leap-15-5"
    sku       = "gen2"
    version   = "latest"
  }
}
