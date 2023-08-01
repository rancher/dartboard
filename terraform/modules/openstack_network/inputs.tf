variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "availability_zone" {
  description = "Availability zone where the instance is created"
}

variable "network_id" {
  description = "The UUID of the parent network where subnet will be created"
}

variable "subnet_cidr" {
  description = "The CIDR range for the subnet that will be created"
}

variable "keypair" {
  description = "Openstack Keypair resource"
}

variable "bastion_flavor" {
  description = "Bastion flavor name"
}

variable "bastion_image" {
  description = "Bastion image ID"
}

variable "dns_nameservers" {
  description = "List of DNS servers used for resolving"
  type = list
}

variable "floating_ip_pool_ext" {
  description = "External network Name"
}

variable "ssh_public_key_path" {
  description = "Path of public ssh key"
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key"
}

variable "external_network_id" {
  description = "External network ID"
}
