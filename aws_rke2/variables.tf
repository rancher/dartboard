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
  default     = 1
}

variable "agent_count" {
  description = "Number of agent nodes in this cluster"
  default     = 0
}

variable "ami" {
  description = "AMI ID for all nodes in this cluster"
  type        = string
}

variable "instance_type" {
  description = "EC2 instance type"
  default     = "t3.large"
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

variable "k8s_api_ssh_tunnel_local_port" {
  description = "Local port for the SSH tunnel to the first server node's Kubernetes API port (6443)"
  type        = number
}

variable "additional_ssh_tunnels" {
  description = "Opens additional SSH tunnels to the first server node"
  type        = list(list(number))
  default     = []
}

variable "sans" {
  description = "Additional Subject Alternative Names"
  type        = list(string)
  default     = []
}

variable "rke2_version" {
  description = "RKE2 version"
  default     = "v1.24.4+rke2r1"
}

variable "max_pods" {
  description = "Maximum number of pods per node"
  default     = 110
}

variable "node_cidr_mask_size" {
  description = "Size of the CIDR mask for nodes. Increase when increasing max_pods so that 2^(32-node_cidr_max_size) > 2 * max_pods"
  default     = 24
}

variable "secret_values" {
  description = "Value from the secrets module"
  type = object({
    client_ca_key          = string
    client_ca_cert         = string
    server_ca_key          = string
    server_ca_cert         = string
    request_header_ca_key  = string
    request_header_ca_cert = string
    master_user_cert       = string
    master_user_key        = string
    cluster_ca_certificate = string
    api_token_string       = string
  })
}
