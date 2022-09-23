variable "project" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "k3s_version" {
  description = "k3s version"
  default = "v1.23.10+k3s1"
}

variable "name" {
  description = "Name of the machine to SSH into"
  type = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access the instance"
  type = string
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible instances"
  default = null
}

variable "client_ca_key" {
  description = "Client CA key"
  type = string
}

variable "client_ca_cert" {
  description = "Client CA certificate"
  type = string
}

variable "server_ca_key" {
  description = "Server CA key"
  type = string
}

variable "server_ca_cert" {
  description = "Server CA certificate"
  type = string
}

variable "request_header_ca_key" {
  description = "Request header CA key"
  type = string
}

variable "request_header_ca_cert" {
  description = "Request header CA certificate"
  type = string
}