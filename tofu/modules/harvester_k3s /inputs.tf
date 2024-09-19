variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
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

variable "image_name" {
  description = "Image name for all VMs in this cluster"
  type        = string
}

variable "image_namespace" {
  description = "Namespace to search for OR upload image, if it does not exist"
  default     = "default"
}

variable "cpu" {
  description = "Number of CPUs to allocate for the VM(s)"
  default     = 2
}

variable "memory" {
  description = "Number of GB of Memory to allocate for the VM(s)"
  default     = 8
}

variable "ssh_keys" {
  description = "List of SSH key names and namespaces to be pulled from Harvester"
  type = list(object({
    name      = string
    namespace = string
  }))
  default = [{}]
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access cluster nodes"
  type        = string
}

variable "ssh_user" {
  description = "User name to use for the SSH connection"
  type        = string
  default     = "root"
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible instances"
  type        = string
  default     = null
}

variable "ssh_bastion_user" {
  description = "User name for the SSH bastion host's OS"
  default     = "root"
}

variable "subnet_id" {
  description = "ID of the subnet to connect to"
  type        = string
}

variable "secondary_subnet_id" {
  description = "ID of the secondary subnet to connect to"
  type        = string
  default     = null
}

variable "vpc_security_group_id" {
  description = "ID of the security group to connect to"
  type        = string
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
  description = "Additional Subject Alternative Names"
  type        = list(string)
  default     = []
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
  type        = string
  default     = null
}

variable "datastore_endpoint" {
  description = "Override datastore with a custom endpoint string"
  type        = string
  default     = null
}

variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default     = ["cat /etc/os-release"]
}
