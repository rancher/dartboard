variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "distro_version" {
  description = "Kubernetes version for AKS to use"
  default     = "1.26.3"
}

variable "server_count" {
  description = "Ignored"
  type        = number
  default     = null
}

variable "agent_count" {
  description = "Number of nodes in this cluster"
  default     = 1
}

variable "reserve_node_for_monitoring" {
  description = "Whether to reserve a node for monitoring. If true, adds a taint and toleration with label 'monitoring' to the first agent node"
  default     = false
}

variable "ssh_private_key_path" {
  description = "Ignored"
  type        = string
  default     = null
}

variable "ssh_user" {
  description = "Ignored"
  type        = string
  default     = null
}

variable "local_kubernetes_api_port" {
  description = "Ignored"
  type        = number
  default     = null
}

variable "tunnel_app_http_port" {
  description = "Ignored"
  type        = number
  default     = null
}

variable "tunnel_app_https_port" {
  description = "Ignored"
  type        = number
  default     = null
}

variable "sans" {
  description = "Ignored"
  type        = list(string)
  default     = null
}

variable "max_pods" {
  description = "Maximum number of pods per node"
  default     = 250
}

variable "node_cidr_mask_size" {
  description = "Ignored"
  type        = number
  default     = null
}

variable "enable_audit_log" {
  description = "Whether to enable audit logging"
  default     = false
}

variable "backend" {
  description = "Ignored"
  type        = string
  default     = null
}

variable "host_backend_variables" {
  description = "Backend-specific configuration variables for all nodes in this cluster"
  type        = any
}

variable "network_backend_variables" {
  description = "Backend-specific configuration variables for the network in this cluster"
  type        = any
}
