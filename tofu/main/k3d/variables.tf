variable "upstream_cluster" {
  description = "Upstream cluster configuration. See tofu/modules/generic/test_environment/variables.tf for details"
  type = any
}

variable "upstream_cluster_distro_module" {
  description = "Name of the module to use for the upstream cluster"
  default     = "k3d/k3s"
}

variable "downstream_cluster_templates" {
  description = "List of downstream cluster configurations. See tofu/modules/generic/test_environment/variables.tf for details"
  type = list(any)
}

variable "downstream_cluster_distro_module" {
  description = "Name of the module to use for the downstream clusters"
  default     = "k3d/k3s"
}

variable "tester_cluster" {
  description = "Tester cluster configuration. See tofu/modules/generic/test_environment/variables.tf for details"
  type = any
  default = null
}

variable "tester_cluster_distro_module" {
  description = "Name of the module to use for the downstream clusters"
  default     = "k3d/k3s"
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
