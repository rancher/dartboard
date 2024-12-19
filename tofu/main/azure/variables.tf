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

variable "upstream_cluster" {
  description = "Upstream cluster configuration. See tofu/modules/generic/test_environment/variables.tf for details"
  type = any
}

variable "upstream_cluster_distro_module" {
  description = "Name of the module to use for the upstream cluster"
  default     = "generic/k3s"
}

variable "downstream_cluster_templates" {
  description = "List of downstream cluster configurations. See tofu/modules/generic/test_environment/variables.tf for details"
  type = list(any)
}

variable "downstream_cluster_distro_module" {
  description = "Name of the module to use for the downstream clusters"
  default     = "generic/k3s"
}

variable "tester_cluster" {
  description = "Tester cluster configuration. See tofu/modules/generic/test_environment/variables.tf for details"
  type = any
  default = null
}

variable "tester_cluster_distro_module" {
  description = "Name of the module to use for the downstream clusters"
  default     = "k3d/k3s"
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
