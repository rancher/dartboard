variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "ssh_user" {
  description = "User name to use for the SSH connection to the host"
  type        = string
  default     = "root"
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access cluster nodes"
  type        = string
}

variable "node_module" {
  description = "Non-generic module to create nodes"
  type        = string
}

variable "node_module_variables" {
  description = "Node module-specific configuration variables for the database host"
  type        = any
}

variable "network_config" {
  description = "Network module outputs, to be passed to node_module"
  type        = any
}

variable "kine_password" {
  description = "Password for the Kine Postgres user"
  type        = string
  default     = "kinepassword"
}
