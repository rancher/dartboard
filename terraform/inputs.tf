locals {
  distro_version = "v1.24.12+k3s1"

  old_upstream_server_count        = 1
  old_upstream_agent_count         = 0
  old_upstream_san                 = "oldupstream.local.gd"
  old_upstream_kubernetes_api_port = 6445
  old_upstream_public_port         = 8443

  old_downstream_server_count        = 1
  old_downstream_agent_count         = 0
  old_downstream_san                 = "olddownstream.local.gd"
  old_downstream_kubernetes_api_port = 6446


  new_upstream_server_count        = 1
  new_upstream_agent_count         = 0
  new_upstream_san                 = "newupstream.local.gd"
  new_upstream_kubernetes_api_port = 6447
  new_upstream_public_port         = 8444

  new_downstream_server_count        = 1
  new_downstream_agent_count         = 0
  new_downstream_san                 = "newdownstream.local.gd"
  new_downstream_kubernetes_api_port = 6448

  project_name = "test"
}
