locals {
  upstream_server_count        = 3
  upstream_agent_count         = 2
  upstream_distro_version      = "v1.24.12+k3s1"
  upstream_san                 = "upstream.local.gd"
  upstream_kubernetes_api_port = 6445

  rancher_chart = "https://releases.rancher.com/server-charts/latest/rancher-2.7.2.tgz"
  rancher_port  = 8443

  downstream_server_count        = 1
  downstream_agent_count         = 0
  downstream_distro_version      = "v1.24.12+k3s1"
  downstream_san                 = "downstream.local.gd"
  downstream_kubernetes_api_port = 6446

  project_name = "moio"
}
