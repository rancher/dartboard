locals {
  region                      = "us-east-1"
  availability_zone           = "us-east-1a"
  secondary_availability_zone = "us-east-1b"

  bastion_ami = "ami-0abac89b48b8cc3bb" // amazon/suse-sles-15-sp4-byos-v20220621-hvm-ssd-arm64

  upstream_instance_type       = "t3a.xlarge"
  upstream_ami                 = "ami-0096528c9fcc1a6a9" // Ubuntu: us-east-1 jammy 22.04 LTS amd64 hvm:ebs-ssd 20221118
  upstream_server_count        = 3
  upstream_agent_count         = 0
  upstream_distro_version      = "v1.24.6+k3s1"
  upstream_max_pods            = 110
  upstream_node_cidr_mask_size = 24
  rancher_chart                = "https://releases.rancher.com/server-charts/latest/rancher-2.6.9.tgz"
  upstream_san                 = "upstream.local.gd"
  upstream_local_port          = 6443
  upstream_datastore           = null

  project_name         = "moio"
  ssh_private_key_path = "~/.ssh/id_ed25519"
  ssh_public_key_path  = "~/.ssh/id_ed25519.pub"
}
