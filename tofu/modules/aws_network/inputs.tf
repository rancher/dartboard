variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "region" {
  description = "Region where the instance is created"
  type        = string
}

variable "availability_zone" {
  description = "Availability zone where the instance is created"
  type        = string
}

variable "secondary_availability_zone" {
  description = "Optional secondary availability zone. Setting creates of a secondary private subnet"
  type        = string
  default     = null
}

variable "ssh_public_key_path" {
  description = "Path of public ssh key for AWS"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key for AWS"
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

variable "bastion_host_ami" {
  description = "AMI ID"
  default     = "ami-0e55a8b472a265e3f"
  // openSUSE-Leap-15-5-v20230608-hvm-ssd-arm64-a516e959-df54-4035-bb1a-63599b7a6df9
}

variable "bastion_host_instance_type" {
  description = "EC2 instance type"
  default     = "t4g.small"
}
