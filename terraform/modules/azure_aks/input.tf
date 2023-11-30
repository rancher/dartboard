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

variable "default_node_pool_count" {
  description = "Number of nodes in the default (system) pool for this cluster"
  default     = 1
}

variable "secondary_node_pool_count" {
  description = "Number of nodes in this cluster's secondary pool (for workloads)"
  default     = 0
}

variable "secondary_node_pool_labels" {
  description = "Labels to apply to the secondary pool nodes (eg. {key = value})"
  type        = map(string)
  default     = {}
}

variable "secondary_node_pool_taints" {
  description = "Taints to apply to the secondary pool nodes (eg. ['monitoring=true:NoSchedule'])"
  type        = list(string)
  default     = []
}

variable "vm_size" {
  description = "Azure VM instance type of all nodes in this cluster"
  default     = "Standard_B2as_v2"
}

variable "os_disk_type" {
  description = "Provisioned root disk type: 'Ephemeral' or 'Managed'"
  default     = "Managed"
}

variable "os_disk_size" {
  description = " The Size of the Internal OS Disk in GB"
  default     = 30
}

variable "subnet_id" {
  description = "Azure Subnet id to attach the VM NIC"
  type        = string
}

variable "distro_version" {
  description = "Kubernetes version for AKS to use"
  default     = "1.26.3"
}
