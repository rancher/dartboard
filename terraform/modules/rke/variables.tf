variable "project" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "distro_version" {
  description = "RKE version followed by the Kubernetes version, see https://github.com/rancher/rke/releases"
  default     = "v1.3.15/rke_darwin-amd64 v1.23.10-rancher1-1"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "server_names" {
  description = "List of names of server nodes (controlplane + etcd) to deploy"
  type        = list(string)
}

variable "agent_names" {
  description = "List of names of agent nodes (worker) to deploy"
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
