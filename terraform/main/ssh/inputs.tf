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
  description = "User name for SSH access"
  default     = "root"
}

variable "nodes" {
  description = "Node names and FQDNs in per-cluster lists, see terraform/examples/ssh.tfvars"
  type        = list(list(object({ fqdn : string, name : string })))
}
