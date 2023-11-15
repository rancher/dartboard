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

variable "instance_type" {
  description = "Azure VM instance type"
  default     = "Standard_B2as_v2"
}

variable "admin_username" {
  description = "Azure VM admin user name"
  default     = "azureuser"
}

variable "subnet_id" {
  description = "Azure Subnet id to attach the VM NIC"
  type        = string
}

variable "public_ip_address_id" {
  description = "Public IP to attach to the VM, optional"
  default     = null
}

# TODO: commands should be actually applied
variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default     = ["cat /etc/os-release"]
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