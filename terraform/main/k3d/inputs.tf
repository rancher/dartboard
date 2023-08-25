locals {
  project_name = "st"

  upstream_cluster = {
    name           = "upstream"
    server_count   = 1
    agent_count    = 0
    distro_version = "v1.24.12+k3s1"
    agent_labels   = []
    agent_taints   = []

    // k3d-specific
    local_name = "upstream.local.gd"
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

      // k3d-specific
      local_name = "downstream-${i}.local.gd"
    }
  ]

  tester_cluster = {
    name           = "tester"
    server_count   = 1
    agent_count    = 0
    distro_version = "v1.24.12+k3s1"
    agent_labels   = []
    agent_taints   = []

    // k3d-specific
    local_name = "tester.local.gd"
  }

  clusters = concat([local.upstream_cluster], local.downstream_clusters, [local.tester_cluster])

  // k3d-specific
  first_local_kubernetes_api_port = 6445
  first_local_http_port           = 8080
  first_local_https_port          = 8443
}
