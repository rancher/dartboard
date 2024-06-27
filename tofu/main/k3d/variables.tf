# Frequently changed variables

variable "downstream_cluster_count" {
  description = "Number of downstream clusters"
  default     = 0
}

variable "distro_version" {
  description = "Version of the Kubernetes distro in all clusters"
  default     = "v1.26.9+k3s1"
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


# Downstream cluster specifics

variable "downstream_server_count" {
  description = "Number of server nodes in each downstream cluster"
  default     = 1
}

variable "downstream_agent_count" {
  description = "Number of agent nodes in the downstream cluster"
  default     = 0
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


# "Multi-tenancy" variables

variable "project_name" {
  description = "Name of this project, used as prefix for resources it creates"
  default     = "st"
}

variable "first_kubernetes_api_port" {
  description = "Port number where the Kubernetes API of the first cluster is published locally. Other clusters' ports are published in successive ports"
  default     = 6445
}

variable "first_app_http_port" {
  description = "Port number where the first server's port 80 is published locally. Other clusters' ports are published in successive ports"
  default     = 8080
}

variable "first_app_https_port" {
  description = "Port number where the first server's port 443 is published locally. Other clusters' ports are published in successive ports"
  default     = 8443
}
