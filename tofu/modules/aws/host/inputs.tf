variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "availability_zone" {
  description = "Availability zone where the instance is created"
  type        = string
}

variable "name" {
  description = "Symbolic name of this instance"
  type        = string
}

variable "ami" {
  description = "AMI ID"
  default     = "ami-0e55a8b472a265e3f"
  // openSUSE-Leap-15-5-v20230608-hvm-ssd-arm64-a516e959-df54-4035-bb1a-63599b7a6df9
}

variable "instance_type" {
  description = "EC2 instance type"
  default     = "t4g.small"
}

variable "ssh_key_name" {
  description = "Name of the SSH key used to access the instance"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access the instance"
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

variable "ssh_tunnels" {
  description = "Opens SSH tunnels to this host via the bastion"
  type        = list(list(number))
  default     = []
}

variable "subnet_id" {
  description = "ID of the subnet to connect to"
  type        = string
}

variable "vpc_security_group_id" {
  description = "ID of the security group to connect to"
  type        = string
}

variable "root_volume_size_gb" {
  description = "Size of the root volume"
  default     = 50
}

variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default     = ["cat /etc/os-release"]
}
