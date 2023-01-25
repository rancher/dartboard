variable "upstream_datastore" {
  type = string
}

variable "kine_image" {
  type = string
}

locals {
  upstream_server_count   = 3
  upstream_agent_count    = 0
  upstream_distro_version = "v1.24.6+k3s1"
  upstream_san            = "upstream.local.gd"

  project_name = "moio"
}
