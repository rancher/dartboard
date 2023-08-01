variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this cluster"
}

variable "availability_zone" {
  description = "Availability zone where the instance is created"
}

variable "server_count" {
  description = "Number of server nodes in this cluster"
  default     = 1
}

variable "agent_count" {
  description = "Number of agent nodes in this cluster"
  default     = 0
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

variable "floating_ip_pool_ext" {
  description = "External network Name"
}

variable "flavor_name" {
  description = "Bastion flavor name"
}

variable "image_id" {
  description = "Bastion image id"
}

variable "keypair" {
  description = "Name of the SSH key used to access cluster nodes"
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access cluster nodes"
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible nodes"
  default     = null
}

variable "network_id" {
  description = "Private network ID"
}

variable "subnet_id" {
  description = "Subnet of the Private Network ID"
}

variable "distro_version" {
  description = "k3s version"
  default     = "v1.23.10+k3s1"
}

variable "max_pods" {
  description = "Maximum number of pods per node"
  default     = 110
}

variable "node_cidr_mask_size" {
  description = "Size of the CIDR mask for nodes. Increase when increasing max_pods so that 2^(32-node_cidr_max_size) > 2 * max_pods"
  default     = 24
}

variable "datastore" {
  description = "Data store to use: mariadb, postgres or leave for a default (sqlite for one-server-node installs, embedded etcd otherwise)"
  default     = null
}

variable "datastore_endpoint" {
  description = "Override datastore with a custom endpoint string"
  default     = null
}

variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default     = ["cat /etc/os-release"]
}
