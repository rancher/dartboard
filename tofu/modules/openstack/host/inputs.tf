variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "availability_zone" {
  description = "Availability zone where the instance is created"
  default     = "nova"
}

variable "name" {
  description = "Symbolic name of this instance"
}

variable "image" {
  description = "OpenStack Image/Glance ID"
  default     = "af0caca5-fd24-4340-a84a-e60e088bc92d" // OVHcloud: GRA7 - CentOS7
}

variable "flavor" {
  description = "OpenStack Flavor name"
  default     = "b2-7"
}

variable "keypair" {
  description = "OpenStack Keypair name"
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible instances"
  type        = string
  default     = null
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access the instance"
  type        = string
}

variable "network_id" {
  description = "ID of the Network to connect to"
}

variable "subnet_id" {
  description = "ID of the subnet to connect to"
}

variable "attach_floating_ip_from" {
  description = "Network name to spawn a Floating IP for this host. Should be a Public IP address. Leve null if there is no need for public exposition"
  default     = null
}

variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default     = ["cat /etc/os-release"]
}

variable "ip_wildcard_resolver_domain" {
  description = "Wildcard resolver to craft DNS name to an IP address"
  default     = "nip.io"
}
