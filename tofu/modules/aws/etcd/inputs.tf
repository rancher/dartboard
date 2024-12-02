variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "availability_zone" {
  description = "Availability zone where the instance is created"
  type        = string
}

variable "server_count" {
  description = "Number of server nodes in this cluster"
  default     = 3
}

variable "ami" {
  description = "AMI ID for all nodes in this cluster"
  type        = string
}

// see https://etcd.io/docs/v3.5/op-guide/hardware/
variable "instance_type" {
  description = "EC2 instance type"
  default     = "t3a.large"
}

variable "ssh_key_name" {
  description = "Name of the SSH key used to access cluster nodes"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access cluster nodes"
  type        = string
}

variable "ssh_user" {
  description = "User name to use for the SSH connection"
  type        = string
  default     = "root"
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible instances"
  type        = string
  default     = null
}

variable "ssh_bastion_user" {
  description = "User name for the SSH bastion host's OS"
  default     = "root"
}

variable "subnet_id" {
  description = "ID of the subnet to connect to"
  type        = string
}

variable "vpc_security_group_id" {
  description = "ID of the security group to connect to"
  type        = string
}

variable "additional_ssh_tunnels" {
  description = "Opens additional SSH tunnels to the first server node"
  type        = list(list(number))
  default     = []
}

variable "etcd_version" {
  description = "etcd version"
  default     = "v3.5.6"
}
