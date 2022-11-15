// AMI lookups

data "aws_ami" "sles15sp4" {
  most_recent = true
  name_regex  = "^suse-sles-15-sp4-byos-v"
  owners      = ["013907871322"]

  filter {
    name   = "architecture"
    values = ["arm64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }
}

data "aws_ami" "rocky8" {
  most_recent = true
  name_regex  = "Rocky-8"
  owners      = ["792107900819"]

  filter {
    name   = "architecture"
    values = ["x86_64"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }
}

resource "aws_key_pair" "key_pair" {
  key_name   = "${var.project_name}-key-pair"
  public_key = file(var.ssh_public_key_path)

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-ssh-key-pair"
  }
}

output "key_name" {
  value = aws_key_pair.key_pair.key_name
}

output "latest_sles_ami" {
  value = data.aws_ami.sles15sp4
}

output "latest_rocky_ami" {
  value = data.aws_ami.rocky8
}
