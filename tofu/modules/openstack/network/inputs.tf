variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "availability_zone" {
  description = "Availability zone where hosts are created"
  type        = string
}

variable "network_id" {
  description = "The UUID of the parent network where subnet will be created"
  type        = string
}

variable "subnet_cidr" {
  description = "The CIDR range for the subnet that will be created"
  type        = string
}

variable "keypair" {
  description = "OpenStack Keypair resource"
  type        = string
}

variable "bastion_flavor" {
  description = "Bastion flavor name"
  type        = string
}

variable "bastion_image" {
  description = "Bastion image ID"
  type        = string
}

variable "dns_nameservers" {
  description = "List of DNS servers used for resolving"
  type        = list(string)
}

variable "floating_ip_pool_ext" {
  description = "External network Name"
  type        = string
}

variable "ssh_public_key_path" {
  description = "Path of public ssh key"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key"
  type        = string
}

variable "external_network_id" {
  description = "External network ID"
  type        = string
}
