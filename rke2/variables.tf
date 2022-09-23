variable "project" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "rke2_version" {
  description = "RKE2 version"
  default = "v1.24.4+rke2r1"
}


variable "server_names" {
  description = "List of names of server nodes to deploy"
  type = list(string)
}

variable "agent_names" {
  description = "List of names of agent nodes to deploy"
  type = list(string)
  default = []
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access the instance"
  type = string
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible instances"
  default = null
}

variable "max_pods" {
  description = "Maximum number of pods per node"
  default = 110
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
