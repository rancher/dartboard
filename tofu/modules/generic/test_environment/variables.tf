variable "ssh_public_key_path" {
  description = "Path to SSH public key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519.pub"
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519"
}

variable "ssh_user" {
  description = "User name to use for the SSH connection to all nodes in all clusters"
  default     = "root"
}

variable "node_module" {
  description = "Non-generic module to create nodes in clusters for this environment"
  type        = string
}

variable "network_config" {
  description = "Network module outputs, to be passed to node_module"
  type        = any
}

# Upstream cluster specifics
variable "upstream_cluster" {
  type = object({
    server_count   = number // Number of server nodes in the upstream cluster
    agent_count    = number // Number of agent nodes in the upstream cluster
    distro_version = string // Version of the Kubernetes distro in the upstream cluster

    public_ip                   = bool // Whether the upstream cluster should have a public IP assigned
    reserve_node_for_monitoring = bool // Set a 'monitoring' label and taint on one node of the upstream cluster to reserve it for monitoring
    enable_audit_log            = bool // Enable audit log for the cluster

    node_module_variables = any // Node module-specific variables
  })
}

variable "upstream_cluster_distro_module" {
  description = "Name of the module to use for the upstream cluster"
  default     = "generic/k3s"
}

# Downstream cluster specifics
variable "downstream_cluster_templates" {
  type = list(object({
    cluster_count  = number // Number of downstream clusters that should be created using this configuration
    server_count   = number // Number of server nodes in the downstream cluster
    agent_count    = number // Number of agent nodes in the downstream cluster
    distro_version = string // Version of the Kubernetes distro in the downstream cluster

    public_ip                   = bool // Whether the downstream cluster should have a public IP assigned. Default false
    reserve_node_for_monitoring = bool // Set a 'monitoring' label and taint on one node of the downstream cluster to reserve it for monitoring. Default false
    enable_audit_log            = bool // Enable audit log for the cluster. Default false

    node_module_variables = any // Node module-specific variables
  }))
}

# Note: this is kept constant for all templates because OpenTofu v1.8.2 does not allow to use
# each.value, each.key or count.index in expressions for module paths
# context is https://github.com/opentofu/opentofu/blob/main/rfc/20240513-static-evaluation/module-expansion.md ->
# https://github.com/opentofu/opentofu/issues/1896#issuecomment-2275763570 ->
# https://github.com/opentofu/opentofu/issues/2155
variable "downstream_cluster_distro_module" {
  description = "Name of the module to use for downstream clusters. Default assumes imported cluster"
  default     = "generic/k3s"
}

variable "node_templates" {
  type = list(object({
    node_count              = number // Number of nodes in this configuration
    name_prefix             = string // String to prefix the name of each node in this configuration
    node_module_variables   = any    // Node module-specific variables
  }))
}

# Tester cluster specifics
variable "tester_cluster" {
  type = object({
    server_count   = number // Number of server nodes in the tester cluster
    agent_count    = number // Number of agent nodes in the tester cluster
    distro_version = string // Version of the Kubernetes distro in the tester cluster

    public_ip                   = bool // Whether the tester cluster should have a public IP assigned
    reserve_node_for_monitoring = bool // Set a 'monitoring' label and taint on one node of the tester cluster to reserve it for monitoring
    enable_audit_log            = bool // Enable audit log for the cluster

    node_module_variables = any // Node module-specific variables
  })                            # If null, no tester cluster will be created
  nullable = true
}

variable "tester_cluster_distro_module" {
  description = "Name of the module to use for the tester cluster"
  default     = "generic/k3s"
}

# "Multi-tenancy" variables
variable "project_name" {
  description = "Name of this project, used as prefix for resources it creates"
  default     = "st"
}

variable "first_kubernetes_api_port" {
  description = "Port number where the Kubernetes API of the first cluster is published locally. Other clusters' ports are published in successive ports"
  type        = number
  default     = 7445
}

variable "first_app_http_port" {
  description = "Port number where the first server's port 80 is published locally. Other clusters' ports are published in successive ports"
  type        = number
  default     = 9080
}

variable "first_app_https_port" {
  description = "Port number where the first server's port 443 is published locally. Other clusters' ports are published in successive ports"
  type        = number
  default     = 9443
}
