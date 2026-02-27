variable "ssh_public_key_path" {
  description = "Path to SSH public key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519.pub"
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519"
}

variable "ssh_user" {
  description = "User name to use for the SSH connection to all nodes in all clusters"
  default     = "root"
}

variable "ssh_bastion_user" {
  description = "User name for the SSH bastion host's OS"
  default     = "root"
}

variable "ssh_prefix_list" {
  description = "The name of an existing prefix list of IP addresses approved for SSH access"
  type        = string
  default     = null
}

variable "upstream_cluster" {
  description = "Upstream cluster configuration. See tofu/modules/generic/test_environment/variables.tf for details"
  type        = any
}

variable "upstream_cluster_distro_module" {
  description = "Name of the module to use for the upstream cluster"
  default     = "generic/k3s"
}

variable "downstream_cluster_templates" {
  description = "List of downstream cluster configurations. See tofu/modules/generic/test_environment/variables.tf for details"
  type        = list(any)
}

variable "downstream_cluster_distro_module" {
  description = "Name of the module to use for the downstream clusters"
  default     = "generic/k3s"
}

variable "tester_cluster" {
  description = "Tester cluster configuration. See tofu/modules/generic/test_environment/variables.tf for details"
  type        = any
  default     = null
}

variable "tester_cluster_distro_module" {
  description = "Name of the module to use for the downstream clusters"
  default     = "generic/k3s"
}

# "Multi-tenancy" variables
variable "project_name" {
  description = "Name of this project, used as prefix for resources it creates"
  default     = "st"
}

variable "first_kubernetes_api_port" {
  description = "Port number where the Kubernetes API of the first cluster is published locally. Other clusters' ports are published in successive ports"
  default     = 7445
}

variable "first_app_http_port" {
  description = "Port number where the first server's port 80 is published locally. Other clusters' ports are published in successive ports"
  default     = 9080
}

variable "first_app_https_port" {
  description = "Port number where the first server's port 443 is published locally. Other clusters' ports are published in successive ports"
  default     = 9443
}

# AWS-specific variables
variable "region" {
  description = "AWS region for this deployment"
  default     = "us-east-1"
}

variable "aws_profile" {
  description = "Local ~/.aws/config profile to utilize for AWS access"
  type        = string
  default     = null
}

variable "availability_zone" {
  description = "AWS availability zone for this deployment"
  default     = "us-east-1a"
}

variable "existing_vpc_name" {
  description = "Name of existing VPC to use. If null, a new VPC will be created"
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
  type        = string
  default     = "t4g.large"
}