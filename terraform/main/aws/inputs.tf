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

      // aws-specific
      local_name    = "upstream.local.gd"
      instance_type = "i3.large"
      ami           = "ami-009fd8a4732ea789b" // openSUSE-Leap-15-5-v20230608-hvm-ssd-x86_64
    },
    {
      name           = "downstream"
      server_count   = 1
      agent_count    = 0
      distro_version = "v1.24.12+k3s1"
      agent_labels   = []
      agent_taints   = []

      // aws-specific
      local_name    = "downstream.local.gd"
      instance_type = "t4g.large"
      ami           = "ami-0e55a8b472a265e3f" // openSUSE-Leap-15-5-v20230608-hvm-ssd-arm64
    },
    {
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
    },
  ]

  // aws-specific
  first_local_kubernetes_api_port = 7445
  first_local_http_port           = 9080
  first_local_https_port          = 9443
  region                          = "us-east-1"
  availability_zone               = "us-east-1a"
  ssh_private_key_path            = "~/.ssh/id_ed25519"   // generate with `ssh-keygen -t ed25519`
  ssh_public_key_path             = "~/.ssh/id_ed25519.pub"
}
