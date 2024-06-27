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

variable "ssh_public_key_path" {
  description = "Path of public ssh key for AWS"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key for AWS"
  type        = string
}
