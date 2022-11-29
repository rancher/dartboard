locals {
  region                      = "us-east-1"
  availability_zone           = "us-east-1a"
  secondary_availability_zone = "us-east-1b"

  bastion_ami = "ami-0abac89b48b8cc3bb" // amazon/suse-sles-15-sp4-byos-v20220621-hvm-ssd-arm64

  upstream_instance_type       = "t2.medium"
  upstream_ami                 = "ami-04fc00d791d804b24" // Ubuntu: us-east-1 bionic 18.04 LTS amd64 hvm:ebs-ssd 20220926
  upstream_server_count        = 1
  upstream_agent_count         = 0
  upstream_distro_version      = "v1.3.11/rke_darwin-amd64 v1.22.9-rancher1-1"
  upstream_max_pods            = 300
  upstream_node_cidr_mask_size = 22
  rancher_chart                = "https://releases.rancher.com/server-charts/latest/rancher-2.6.6.tgz"
  upstream_san                 = "upstream.local.gd"
  upstream_local_port          = 6443

  downstream_instance_type       = "t3.medium"
  downstream_ami                 = "ami-0746c2106d76fa617" // 792107900819/Rocky-8-ec2-8.6-20220515.0.x86_64
  downstream_server_count        = 1
  downstream_agent_count         = 0
  downstream_distro_version      = "v1.22.13+rke2r1"
  downstream_max_pods            = 300
  downstream_node_cidr_mask_size = 22
  downstream_san                 = "downstream.local.gd"
  downstream_local_port          = 6444

  project_name         = "moio"
  ssh_private_key_path = "~/.ssh/id_ed25519"
  ssh_public_key_path  = "~/.ssh/id_ed25519.pub"
}
