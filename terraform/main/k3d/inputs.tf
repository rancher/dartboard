locals {
  project_name = "moio"

  clusters = [
    {
      name           = "upstream"
      server_count   = 3
      agent_count    = 2
      distro_version = "v1.24.12+k3s1"
      agent_labels   = [
        [{ key : "monitoring", value : "true" }]
      ]
      agent_taints = [
        [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
      ]
      local_name = "upstream.local.gd"
    },
    {
      name           = "downstream"
      server_count   = 1
      agent_count    = 0
      distro_version = "v1.24.12+k3s1"
      agent_labels   = []
      agent_taints   = []
      local_name     = "downstream.local.gd"
    },
    {
      name           = "tester"
      server_count   = 1
      agent_count    = 0
      distro_version = "v1.24.12+k3s1"
      agent_labels   = []
      agent_taints   = []
      local_name     = "tester.local.gd"
    },
  ]

  first_local_kubernetes_api_port = 6445
  first_local_http_port           = 8080
  first_local_https_port          = 8443
}
