variable "project_name" {
  description = "A prefix for names of objects created by this module"
  type        = string
  default     = "st"
}

variable "etcd_version" {
  description = "etcd version"
  type        = string
  default     = "v3.7.1" # change below shasum when updating this
}

variable "etcd_shasum" {
  description = "shasum of the corresponding version of the etcd amd64 release tarball"
  type        = string
  default     = "e8cd3fa8064c98137c5dbd78b76f969417ace84efb83c481041d7a52ffdd8fb9" # for version 3.7.1
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "server_count" {
  description = "Number of server nodes in this cluster"
  type        = number
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
