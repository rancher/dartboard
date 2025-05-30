variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "etcd_version" {
  description = "etcd version"
  default     = "v3.5.21"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "server_count" {
  description = "Number of server nodes in this cluster"
  default     = 3
}

variable "ssh_user" {
  description = "User name to use for the SSH connection to the host"
  type        = string
  default     = "root"
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access the instance"
  type        = string
}

variable "additional_ssh_tunnels" {
  description = "Opens additional SSH tunnels to the first server node"
  type        = list(list(number))
  default     = []
}

variable "node_module" {
  description = "Non-generic module to create nodes"
  type        = string
}

variable "node_module_variables" {
  description = "Node module-specific configuration variables for all nodes in this cluster"
  type        = any
}

variable "network_config" {
  description = "Network module outputs, to be passed to node_module"
  type        = any
}
