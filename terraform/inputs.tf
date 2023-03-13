locals {
  region                      = "us-east-1"
  availability_zone           = "us-east-1a"
  secondary_availability_zone = "us-east-1b"

  bastion_ami = "ami-0b46b6463b1f19e83" // suse-sles-15-sp4-byos-v20221216-hvm-ssd-arm64-636dec81-77a2-4706-9fc7-6fa1c294d759

  upstream_instance_type       = "t4g.large"
  upstream_ami                 = "ami-0c6c29c5125214c77" // Ubuntu: us-east-1 jammy 22.04 LTS amd64 hvm:ebs-ssd 20230303
  upstream_server_count        = 3
  upstream_agent_count         = 0
  upstream_distro_version      = "v1.24.10+k3s1"
  rancher_chart                = "https://releases.rancher.com/server-charts/latest/rancher-2.7.1.tgz"
  upstream_san                 = "upstream.local.gd"
  upstream_kubernetes_api_port = 6443

  project_name         = "moio"
  ssh_private_key_path = "~/.ssh/id_ed25519"
  ssh_public_key_path  = "~/.ssh/id_ed25519.pub"
}
