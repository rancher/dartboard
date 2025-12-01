variable "project_name" {
  description = "A prefix for names of objects created by this module"
  type        = string
  default     = "st"
}

variable "location" {
  description = "Azure Location where the instance in created"
  type        = string
}

variable "ssh_bastion_user" {
  description = "User name to use for the SSH connection to the bastion host"
  type        = string
  default     = "azureuser"
}

variable "ssh_public_key_path" {
  description = "Path of public ssh key for hosts created by this module"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh for hosts created by this module"
  type        = string
}

variable "tags" {
  description = "Tags to be applied to all resources created by this module"
  type        = map(string)
  default = {
    Owner = "st"
  }
}

variable "bastion_os_image" {
  description = "OS image to use for the bastion host"
  type = object({
    publisher = string
    offer     = string
    sku       = string
    version   = string
  })
  default = {
    publisher = "suse"
    offer     = "opensuse-leap-15-5"
    sku       = "gen2"
    version   = "latest"
  }
}
