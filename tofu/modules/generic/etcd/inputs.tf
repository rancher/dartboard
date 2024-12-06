variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "etcd_version" {
  description = "etcd version"
  default     = "v3.5.6"
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

variable "backend" {
  description = "Backend for this cluster"
  type        = string
}

variable "backend_variables" {
  description = "Backend-specific configuration variables for all nodes in this cluster"
  type = any
}

variable "backend_network_variables" {
  description = "Backend-specific configuration variables for the network in this cluster"
  type = any
}
