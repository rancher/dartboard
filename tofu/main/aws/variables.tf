# Frequently changed variables

variable "downstream_cluster_count" {
  description = "Number of downstream clusters"
  default     = 0
}

variable "region" {
  description = "AWS region for this deployment"
  default     = "us-east-1"
}

variable "availability_zone" {
  description = "AWS availability zone for this deployment"
  default     = "us-east-1a"
}

variable "ssh_public_key_path" {
  description = "Path to SSH public key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519.pub"
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519"
}

# Upstream cluster specifics

variable "upstream_server_count" {
  description = "Number of server nodes in the upstream cluster"
  default     = 1
}

variable "upstream_agent_count" {
  description = "Number of agent nodes in the upstream cluster"
  default     = 0
}

variable "upstream_reserve_node_for_monitoring" {
  description = "Set a 'monitoring' label and taint on one node of the upstream cluster to reserve it for monitoring"
  default = false
}

variable "upstream_distro_version" {
  description = "Version of the Kubernetes distro in the upstream cluster"
  default     = "v1.26.9+k3s1"
}

variable "upstream_public_ip" {
  description = "Whether the upstream cluster should have a public IP assigned"
  default = true
}

variable "upstream_instance_type" {
  description = "Instance type for the upstream cluster nodes"
  default = "i3.large"
}

variable "upstream_ami" {
  description = "AMI for upstream cluster nodes"
  default = "ami-009fd8a4732ea789b" // openSUSE-Leap-15-5-v20230608-hvm-ssd-x86_64
}


# Downstream cluster specifics

variable "downstream_server_count" {
  description = "Number of server nodes in each downstream cluster"
  default     = 1
}

variable "downstream_agent_count" {
  description = "Number of agent nodes in the downstream cluster"
  default     = 0
}

variable "downstream_distro_version" {
  description = "Version of the Kubernetes distro in the downstream clusters"
  default     = "v1.26.9+k3s1"
}

variable "downstream_public_ip" {
  description = "Whether the downstream cluster should have a public IP assigned"
  default = false
}

variable "downstream_instance_type" {
  description = "Instance type for the downstream cluster nodes"
  default = "t4g.large"
}

variable "downstream_ami" {
  description = "AMI for downstream cluster nodes"
  default = "ami-0e55a8b472a265e3f" // openSUSE-Leap-15-5-v20230608-hvm-ssd-arm64
}


# Tester cluster specifics

variable "deploy_tester_cluster" {
  description = "Use false not to deploy a tester cluster"
  default     = true
}

variable "tester_server_count" {
  description = "Number of server nodes in each tester cluster"
  default     = 1
}

variable "tester_agent_count" {
  description = "Number of agent nodes in the tester cluster"
  default     = 0
}

variable "tester_distro_version" {
  description = "Version of the Kubernetes distro in the tester clusters"
  default     = "v1.26.9+k3s1"
}

variable "tester_public_ip" {
  description = "Whether the tester cluster should have a public IP assigned"
  default = true
}

variable "tester_instance_type" {
  description = "Instance type for the tester cluster nodes"
  default = "t3a.large"
}

variable "tester_ami" {
  description = "AMI for tester cluster nodes"
  default = "ami-009fd8a4732ea789b" // openSUSE-Leap-15-5-v20230608-hvm-ssd-x86_64
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
