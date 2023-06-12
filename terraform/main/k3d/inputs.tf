locals {
  project_name = "moio"

  clusters = {
    upstream : {
      server_count   = 3
      agent_count    = 2
      distro_version = "v1.24.12+k3s1"
      agent_labels   = [
        [{ key : "monitoring", value : "true" }]
      ]
      agent_taints = [
        [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
      ]
      san                 = "upstream.local.gd"
      kubernetes_api_port = 6445
      public_http_port    = 8080
      public_https_port   = 8443
    },
    downstream : {
      server_count        = 1
      agent_count         = 0
      distro_version      = "v1.24.12+k3s1"
      agent_labels        = []
      agent_taints        = []
      san                 = "downstream.local.gd"
      kubernetes_api_port = 6446
      public_http_port    = 8081
      public_https_port   = 8444
    },
    tester : {
      server_count        = 1
      agent_count         = 0
      distro_version      = "v1.24.12+k3s1"
      agent_labels        = []
      agent_taints        = []
      san                 = "tester.local.gd"
      kubernetes_api_port = 6447
      public_http_port    = 8082
      public_https_port   = 8445
    },
  }
}
