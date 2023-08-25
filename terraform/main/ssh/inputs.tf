locals {
  project_name = "st"

  upstream_cluster = {
    name           = "upstream"
    server_count   = 1
    agent_count    = 2
    distro_version = "v1.24.12+k3s1"
    agent_labels = [
      [{ key : "monitoring", value : "true" }]
    ]
    agent_taints = [
      [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
    ]
  }

  downstream_clusters = [
    for i in range(1) :
    {
      name           = "downstream-${i}"
      server_count   = 1
      agent_count    = 0
      distro_version = "v1.24.12+k3s1"
      agent_labels   = []
      agent_taints   = []
    }
  ]

  tester_cluster = {
    name           = "tester"
    server_count   = 1
    agent_count    = 0
    distro_version = "v1.24.12+k3s1"
    agent_labels   = []
    agent_taints   = []
  }

  clusters = concat([local.upstream_cluster], local.downstream_clusters, [local.tester_cluster])

  // k3s-specific
  first_local_kubernetes_api_port = 7445
  first_local_http_port           = 9080
  first_local_https_port          = 9443
}


variable "ssh_public_key_path" {
  description = "Path to SSH public key file"
  type        = string
  default     = "~/.ssh/st-ed25519.pub"
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key file. (Can be generated with `ssh-keygen -t ed25519`)"
  type        = string
  default     = "~/.ssh/st"
}

variable "ssh_user" {
  description = "Default ssh user for nodes"
  type        = string
  default     = "root"
}

variable "nodes" {
  description = "Node description"
  type        = list(any)
  default     = []
}
