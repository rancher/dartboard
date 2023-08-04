locals {
  project_name = "st"

  upstream_cluster = {
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

    // aws-specific
    local_name    = "upstream.local.gd"
    instance_type = "i3.large"
    ami           = "ami-009fd8a4732ea789b" // openSUSE-Leap-15-5-v20230608-hvm-ssd-x86_64
  }

  downstream_clusters = [
  for i in range(5) :
  {
    name           = "downstream-${i}"
    server_count   = 3
    agent_count    = 7
    distro_version = "v1.24.12+k3s1"
    agent_labels   = []
    agent_taints   = []

    // aws-specific
    local_name    = "downstream-${i}.local.gd"
    instance_type = "t4g.large"
    ami           = "ami-0e55a8b472a265e3f" // openSUSE-Leap-15-5-v20230608-hvm-ssd-arm64
  }
  ]

  tester_cluster = {
    name           = "tester"
    server_count   = 1
    agent_count    = 0
    distro_version = "v1.24.12+k3s1"
    agent_labels   = []
    agent_taints   = []

    // aws-specific
    local_name    = "tester.local.gd"
    instance_type = "t3a.large"
    ami           = "ami-009fd8a4732ea789b" // openSUSE-Leap-15-5-v20230608-hvm-ssd-x86_64
  }

  clusters = concat([local.upstream_cluster], local.downstream_clusters, [local.tester_cluster])

  // aws-specific
  first_local_kubernetes_api_port = 7445
  first_local_http_port           = 9080
  first_local_https_port          = 9443
  region                          = "us-east-1"
  availability_zone               = "us-east-1a"
}


variable ssh_public_key_path {
    description = "Path to SSH public key file, see also variable `ssh_private_key_path`."
    type = string
    default = "~/.ssh/id_ed25519.pub"
}

variable ssh_private_key_path {
    description = "Path to SSH private key file. (Can be generated with `ssh-keygen -t ed25519`)"
    type = string
    default = "~/.ssh/id_ed25519"
}
