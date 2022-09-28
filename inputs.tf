locals {
  region            = "us-east-1"
  availability_zone = "us-east-1a"

  bastion_ami           = "ami-0e7690ca6cb45d2c5" // amazon/suse-sles-15-sp4-byos-v20220621-hvm-ssd-x86_64

  upstream_instance_type = "t3.large"
  upstream_ami = "ami-0746c2106d76fa617" // 792107900819/Rocky-8-ec2-8.6-20220515.0.x86_64
  upstream_server_count        = 3
  upstream_agent_count         = 0
  upstream_rke2_version        = "v1.23.10+rke2r1"
  rancher_chart         = "https://releases.rancher.com/server-charts/latest/rancher-2.6.5.tgz"
  upstream_san = "upstream.local.gd"
  upstream_local_port = 6443

  downstream_instance_type = "t3.xlarge"
  downstream_ami           = "ami-0746c2106d76fa617" // 792107900819/Rocky-8-ec2-8.6-20220515.0.x86_64
  downstream_server_count        = 3
  downstream_agent_count         = 4
  downstream_rke2_version        = "v1.22.13+rke2r1"
  downstream_max_pods            = 250
  downstream_san = "downstream.local.gd"
  downstream_local_port = 6444

  project_name         = "moio"
  ssh_private_key_path = "~/.ssh/id_ed25519"
  ssh_public_key_path  = "~/.ssh/id_ed25519.pub"
}
