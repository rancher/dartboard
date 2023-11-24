variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "location" {
  description = "Azure Location where the instance in created"
  type        = string
}

variable "resource_group_name" {
  description = "Azure Resource Group name to which the instance should belong"
  type        = string
}

variable "name" {
  description = "Symbolic name of this instance"
  type        = string
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
  description = "Azure VM kind"
  default     = "Standard_B2as_v2"
}

// Spot instances can be Deallocated/Deleted but costs 1/10th
// anyway, seems we have a constraint that only 3 cores can be allocated as Spot instances
// causing provisioning of many nodes to fail
variable "is_spot" {
  description = "Wheter the VM should be allocated as a Spot instance (costs 1/10th of regular ones but could be evicted)"
  default     = false
}

variable "subnet_id" {
  description = "Azure Subnet id to attach the VM NIC"
  type        = string
}

variable "public_ip_address_id" {
  description = "Public IP to attach to the VM, optional"
  type        = string
  default     = null
}

variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default     = ["cat /etc/os-release"]
}

variable "ssh_user" {
  description = "Azure VM admin user name used for ssh access"
  default     = "azureuser"
}

variable "ssh_public_key_path" {
  description = "Path of public ssh key for Azure"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key for Azure"
  type        = string
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible instances"
  default     = null
}

variable "ssh_tunnels" {
  description = "Opens SSH tunnels to this host via the bastion"
  type        = list(list(number))
  default     = []
}
