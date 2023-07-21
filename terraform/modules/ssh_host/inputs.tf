variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this instance"
  type        = string
}

variable "ssh_addr" {
  description = "Host SSH address"
  type        = string
}

variable "ssh_user" {
  description = "Host SSH address"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access the instance"
  type        = string
}

variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default     = ["cat /etc/os-release"]
}
