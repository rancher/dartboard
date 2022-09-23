locals {
  region            = "us-east-1"
  availability_zone = "us-east-1a"

  bastion_instance_type = "t3.large"
  bastion_ami           = "ami-0e7690ca6cb45d2c5" // amazon/suse-sles-15-sp4-byos-v20220621-hvm-ssd-x86_64
  k3s_version="v1.23.10+k3s1"
  rancher_chart ="https://releases.rancher.com/server-charts/latest/rancher-2.6.5.tgz"

  nodes_instance_type = "t3.xlarge"
  nodes_ami           = "ami-0746c2106d76fa617" // 792107900819/Rocky-8-ec2-8.6-20220515.0.x86_64
  server_nodes = 3
  agent_nodes  = 4
  rke2_version = "v1.23.10+rke2r1"
  max_pods = 250

  project_name         = "moio"
  ssh_private_key_path = "~/.ssh/id_ed25519"
  ssh_public_key_path  = "~/.ssh/id_ed25519.pub"
}
