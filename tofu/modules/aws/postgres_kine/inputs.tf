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

variable "ami" {
  description = "AMI ID for all nodes in this cluster"
  default     = "ami-0728ec0041b1d38ac" // Amazon Linux 2 LTS Arm64 Kernel 5.10 AMI 2.0.20221103.3 arm64 HVM gp2
}

variable "instance_type" {
  description = "EC2 instance type"
  default     = "t4g.xlarge"
}

variable "ssh_key_name" {
  description = "Name of the SSH key used to access cluster nodes"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access cluster nodes"
  type        = string
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible nodes"
  type        = string
  default     = null
}

variable "subnet_id" {
  description = "ID of the subnet to connect to"
  type        = string
}

variable "vpc_security_group_id" {
  description = "ID of the security group to connect to"
  type        = string
}

variable "gogc" {
  description = "Tunable parameter for Go's garbage collection, see: https://tip.golang.org/doc/gc-guide"
  type        = number
  default     = null
}

variable "kine_version" {
  description = "Kine version"
  default     = "v0.9.8"
}

variable "kine_executable" {
  description = "Overrides kine_version by copying an executable from this path"
  type        = string
  default     = null
}
