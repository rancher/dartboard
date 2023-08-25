variable "project" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "distro_version" {
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

variable "agent_labels" {
  description = "Per-agent-node lists of labels to apply"
  type        = list(list(object({ key : string, value : string })))
  default     = []
}

variable "agent_taints" {
  description = "Per-agent-node lists of taints to apply"
  type        = list(list(object({ key : string, value : string, effect : string })))
  default     = []
}

variable "sans" {
  description = "Additional Subject Alternative Names"
  type        = list(string)
  default     = []
}

variable "ssh_user" {
  description = "User name for SSH access"
  default     = "root"
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access the instance"
  type        = string
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible instances"
  default     = null
}

variable "local_kubernetes_api_port" {
  description = "Port this cluster's Kubernetes API will be published to (for inclusion in kubeconfig)"
  default     = 6443
}

variable "max_pods" {
  description = "Maximum number of pods per node"
  default     = 110
}

variable "node_cidr_mask_size" {
  description = "Size of the CIDR mask for nodes. Increase when increasing max_pods so that 2^(32-node_cidr_max_size) > 2 * max_pods"
  default     = 24
}

variable "datastore_endpoint" {
  description = "Configuration string for optional data store"
  default     = null
}

// In general case we need to avoid trying to do any cleanup on the hosts
// For example, for AWS the node or IP address cannot be present and the
// instance will never be used again because it's going to be terminated
// From other side, if the ssh_resource object cannot be accessed, it can
// lock removal of the rest of the dependend resource or generate useless
// error messages at teardown stage.
variable "remove_k3s" {
  description = "Try to remove k3s on destroy"
  type        = bool
  default     = false
}
