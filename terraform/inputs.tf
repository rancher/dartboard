locals {
  upstream_server_count   = 3
  upstream_agent_count    = 0
  upstream_distro_version = "v1.24.6+k3s1"
  rancher_chart           = "https://releases.rancher.com/server-charts/latest/rancher-2.6.9.tgz"
  upstream_san            = "upstream.local.gd"
  upstream_datastore      = "postgres"

  project_name = "moio"
}
