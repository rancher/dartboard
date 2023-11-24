variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "location" {
  description = "Azure Location where the instance in created"
  type        = string
}

variable "resource_group_name" {
  description = "Azure Resource Group name to which the instance should belong"
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

variable "os_image" {
  description = "Azure VM OS image"
  type = object({
    publisher = string
    offer     = string
    sku       = string
    version   = string
  })
}

variable "size" {
  description = "Azure VM size"
  default     = "Standard_B2as_v2"
}

variable "ssh_user" {
  description = "Azure VM admin user name used for ssh access"
  default     = "azureuser"
}

variable "ssh_public_key_path" {
  description = "Path to the (public) SSH key used to access cluster nodes"
}

variable "ssh_private_key_path" {
  description = "Path to the private SSH key used to access cluster nodes"
  type        = string
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible nodes."
  default     = null
}

variable "subnet_id" {
  description = "Azure Subnet id to attach the VM NIC"
  type        = string
}

variable "local_kubernetes_api_port" {
  description = "Local port this cluster's Kubernetes API will be published to (via SSH tunnel)"
  default     = 6445
}

variable "local_http_port" {
  description = "Local port this cluster's http endpoints will be published to (via SSH tunnel)"
  default     = 8080
}

variable "local_https_port" {
  description = "Local port this cluster's https endpoints will be published to (via SSH tunnel)"
  default     = 8443
}

variable "sans" {
  description = "Additional Subject Alternative Names"
  type        = list(string)
  default     = []
}
variable "distro_version" {
  description = "RKE2 version"
  default     = "v1.24.4+rke2r1"
}

variable "max_pods" {
  description = "Maximum number of pods per node"
  default     = 110
}

variable "node_cidr_mask_size" {
  description = "Size of the CIDR mask for nodes. Increase when increasing max_pods so that 2^(32-node_cidr_max_size) > 2 * max_pods"
  default     = 24
}

variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default     = ["cat /etc/os-release"]
}