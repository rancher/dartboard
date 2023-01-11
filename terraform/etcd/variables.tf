variable "project" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "etcd_version" {
  description = "etcd version"
  default     = "v3.5.6"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "server_names" {
  description = "List of names of server nodes to deploy"
  type        = list(string)
}

variable "server_ips" {
  description = "List of IP addresses corresponding to server_names"
  type        = list(string)
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access the instance"
  type        = string
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible instances"
  default     = null
}
