# Frequently changed variables
variable "region" {
  description = "AWS region for this deployment"
  default     = "us-east-1"
}

variable "aws_profile" {
  description = "Local ~/.aws/config profile to utilize for AWS access"
  default = ""
}

variable "availability_zone" {
  description = "AWS availability zone for this deployment"
  default     = "us-east-1a"
}

variable "bastion_host_ami" {
  description = "AMI ID"
  default     = "ami-0e55a8b472a265e3f"
  // openSUSE-Leap-15-5-v20230608-hvm-ssd-arm64-a516e959-df54-4035-bb1a-63599b7a6df9
}

variable "ssh_public_key_path" {
  description = "Path to SSH public key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519.pub"
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519"
}

variable "ssh_user" {
  description = "User name to use for the SSH connection"
  type        = string
  default     = "root"
}

variable "ssh_bastion_user" {
  description = "User name for the SSH bastion host's OS"
  default     = "root"
}

# Upstream cluster specifics
variable "upstream_cluster" {
  type = object({
    name_prefix    = string // Prefix to append to objects created for this cluster
    server_count   = number // Number of server nodes in the upstream cluster
    agent_count    = number // Number of agent nodes in the upstream cluster
    distro_version = string // Version of the Kubernetes distro in the upstream cluster

    public_ip = bool // Whether the upstream cluster should have a public IP assigned
    reserve_node_for_monitoring = bool // Set a 'monitoring' label and taint on one node of the upstream cluster to reserve it for monitoring

    // aws-specific
    datastore     = optional(string) // Optional datastore to utilize for the upstream cluster
    instance_type = string // Instance type for the upstream cluster nodes
    instance_tags = map(string) // tags to apply to the EC2 instances
    ami           = string // AMI for upstream cluster nodes
  })
  default = {
    name_prefix    = "upstream"
    server_count   = 1
    agent_count    = 0
    distro_version = "v1.26.9+k3s1"
    public_ip = true
    reserve_node_for_monitoring = false

    // aws-specific
    instance_type = "i3.large"
    instance_tags = {
      "Owner": "st",
      "DoNotDelete": "true"
    }
    ami           = "ami-009fd8a4732ea789b" // openSUSE-Leap-15-5-v20230608-hvm-ssd-x86_64
  }
}

# Downstream cluster specifics
variable "downstream_cluster_templates" {
  type = list(object({
    cluster_count       = number // Number of downstream clusters that should be created using this configuration
    name_prefix    = string // Prefix to append to objects created for this cluster
    server_count   = number // Number of server nodes in the downstream cluster
    agent_count    = number // Number of agent nodes in the downstream cluster
    distro_version = string // Version of the Kubernetes distro in the downstream cluster

    public_ip = bool // Whether the downstream cluster should have a public IP assigned
    reserve_node_for_monitoring = bool // Set a 'monitoring' label and taint on one node of the downstream cluster to reserve it for monitoring

    // aws-specific
    datastore     = optional(string) // Optional datastore to utilize for the downstream cluster
    instance_type = string // Instance type for the downstream cluster nodes
    instance_tags = map(string) // tags to apply to the EC2 instances
    ami           = string // AMI for downstream cluster nodes
  }))
  default = [{
    cluster_count       = 0 // defaults to 0 to keep in-line with previous behavior
    name_prefix    = "downstream"
    server_count   = 1
    agent_count    = 0
    distro_version = "v1.26.9+k3s1"
    public_ip = false
    reserve_node_for_monitoring = false

    // aws-specific
    instance_type = "t4g.large"
    instance_tags = {
      "Owner": "st",
      "DoNotDelete": "true"
    }
    ami           = "ami-0e55a8b472a265e3f" // openSUSE-Leap-15-5-v20230608-hvm-ssd-arm64
  }]
}

# Tester cluster specifics
variable "tester_cluster" {
  type = object({
    name_prefix    = string // Prefix to append to objects created for this cluster
    server_count   = number // Number of server nodes in the tester cluster
    agent_count    = number // Number of agent nodes in the tester cluster
    distro_version = string // Version of the Kubernetes distro in the tester cluster

    public_ip = bool // Whether the tester cluster should have a public IP assigned
    reserve_node_for_monitoring = bool // Set a 'monitoring' label and taint on one node of the tester cluster to reserve it for monitoring

    // aws-specific
    datastore     = optional(string) // Optional datastore to utilize for the tester cluster
    instance_type = string // Instance type for the tester cluster nodes
    instance_tags = map(string) // tags to apply to the EC2 instances
    ami           = string // AMI for tester cluster nodes
  })
  default = {
    name_prefix    = "tester"
    server_count   = 1
    agent_count    = 0
    distro_version = "v1.26.9+k3s1"
    public_ip = true
    reserve_node_for_monitoring = false

    // aws-specific
    instance_type = "t3a.large"
    instance_tags = {
      "Owner": "st",
      "DoNotDelete": "true"
    }
    ami           = "ami-009fd8a4732ea789b" // openSUSE-Leap-15-5-v20230608-hvm-ssd-x86_64
  }
}

variable "deploy_tester_cluster" {
  description = "Use false not to deploy a tester cluster"
  default     = true
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
