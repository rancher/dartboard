variable "project_name" {
  description = "A prefix for names of objects created by this module"
  type        = string
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
  description = "Path of public ssh key for hosts created by this module"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh for hosts created by this module"
  type        = string
}

variable "ssh_prefix_list" {
  description = "The name of an existing prefix list of IP addresses approved for SSH access"
  type        = string
  default     = null
}

variable "ssh_bastion_user" {
  description = "User name to use for the SSH connection to the bastion host"
  type        = string
  default     = "root"
}

variable "bastion_host_ami" {
  description = "AMI ID"
  type        = string
  default     = "ami-0e55a8b472a265e3f"
  // openSUSE-Leap-15-5-v20230608-hvm-ssd-arm64-a516e959-df54-4035-bb1a-63599b7a6df9
}

variable "bastion_host_instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t4g.large"
}

# Variables for existing VPC configuration
variable "existing_vpc_name" {
  description = "Name of existing VPC to use. If null, a new VPC will be created"
  type        = string
  default     = null
}
