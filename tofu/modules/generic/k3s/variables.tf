variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "distro_version" {
  description = "k3s version"
  default     = "v1.23.10+k3s1"
}

variable "server_count" {
  description = "Number of server nodes in this cluster"
  default     = 1
}

variable "agent_count" {
  description = "Number of agent nodes in this cluster"
  default     = 0
}

variable "reserve_node_for_monitoring" {
  description = "Whether to reserve a node for monitoring. If true, adds a taint and toleration with label 'monitoring' to the first agent node"
  default     = false
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access cluster nodes"
  type        = string
}

variable "ssh_user" {
  description = "User name to use for the SSH connection to cluster nodes"
  type        = string
  default     = "root"
}

variable "local_kubernetes_api_port" {
  description = "Local port this cluster's Kubernetes API will be published to (via SSH tunnel)"
  default     = 6445
}

variable "tunnel_app_http_port" {
  description = "Local port this cluster's http endpoints will be published to (via SSH tunnel)"
  default     = 8080
}

variable "tunnel_app_https_port" {
  description = "Local port this cluster's https endpoints will be published to (via SSH tunnel)"
  default     = 8443
}

variable "sans" {
  description = "Additional Subject Alternative Names for the cluster('s first server node)"
  type        = list(string)
  default     = []
}

variable "max_pods" {
  description = "Maximum number of pods per node"
  default     = 110
}

variable "node_cidr_mask_size" {
  description = "Size of the CIDR mask for nodes. Increase when increasing max_pods so that 2^(32-node_cidr_max_size) > 2 * max_pods"
  default     = 24
}

variable "enable_audit_log" {
  description = "Ignored for k3s"
  default     = false
}

variable "datastore_endpoint" {
  description = "Override datastore with a custom endpoint string"
  type        = string
  default     = null
}

variable "backend" {
  description = "Backend for this cluster"
  type        = string
}

variable "host_backend_variables" {
  description = "Backend-specific configuration variables for all nodes in this cluster"
  type = any
}

variable "network_backend_variables" {
  description = "Backend-specific configuration variables for the network in this cluster"
  type = any
}
