variable "project" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "k3s_version" {
  description = "k3s version"
  default     = "v1.23.10+k3s1"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "server_names" {
  description = "List of names of server nodes to deploy"
  type        = list(string)
}

variable "agent_names" {
  description = "List of names of agent nodes to deploy"
  type        = list(string)
  default     = []
}

variable "sans" {
  description = "Additional Subject Alternative Names"
  type        = list(string)
  default     = []
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access the instance"
  type        = string
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible instances"
  default     = null
}

variable "ssh_local_port" {
  description = "Local port for the SSH tunnel for the API"
  default     = null
}

variable "max_pods" {
  description = "Maximum number of pods per node"
  default     = 110
}

variable "node_cidr_mask_size" {
  description = "Size of the CIDR mask for nodes. Increase when increasing max_pods so that 2^(32-node_cidr_max_size) > 2 * max_pods"
  default     = 24
}
