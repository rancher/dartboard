/*
  This module sets up a class B VPC sliced into three subnets, one public and one or two private.
  The public network has an Internet Gateway and accepts SSH connections only.
  The private networks have Internet access but do not accept any connections.
  A secondary private connection is optional, and is used to support RDS use cases.
*/

resource "aws_vpc" "main" {
  cidr_block           = "172.16.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-vpc"
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = local.vpc_id

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-internet-gateway"
  }
}

locals {
  vpc_id         = aws_vpc.main.id
  vpc_cidr_block = aws_vpc.main.cidr_block
}

resource "aws_eip" "nat_eip" {
  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-nat-eip"
  }
}

resource "aws_nat_gateway" "nat" {
  allocation_id = aws_eip.nat_eip.id
  subnet_id     = aws_subnet.public.id

  depends_on = [aws_internet_gateway.main]

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-nat"
  }
}

resource "aws_route_table" "public" {
  vpc_id = local.vpc_id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main.id
  }

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-public-route-table"
  }
}

resource "aws_main_route_table_association" "vpc_internet" {
  vpc_id         = local.vpc_id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table" "private" {
  vpc_id = local.vpc_id

  route {
    cidr_block     = "0.0.0.0/0"
    nat_gateway_id = aws_nat_gateway.nat.id
  }

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-private-route-table"
  }
}

resource "aws_subnet" "public" {
  availability_zone       = var.availability_zone
  vpc_id                  = local.vpc_id
  cidr_block              = "172.16.0.0/24"
  map_public_ip_on_launch = true

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-public-subnet"
  }
}

resource "aws_route_table_association" "public" {
  subnet_id      = aws_subnet.public.id
  route_table_id = aws_route_table.public.id
}

resource "aws_subnet" "private" {
  availability_zone       = var.availability_zone
  vpc_id                  = local.vpc_id
  cidr_block              = "172.16.1.0/24"
  map_public_ip_on_launch = false

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-private-subnet"
  }
}

resource "aws_route_table_association" "private" {
  subnet_id      = aws_subnet.private.id
  route_table_id = aws_route_table.private.id
}

resource "aws_subnet" "secondary_private" {
  count                   = var.secondary_availability_zone != null ? 1 : 0
  availability_zone       = var.secondary_availability_zone
  vpc_id                  = local.vpc_id
  cidr_block              = "172.16.2.0/24"
  map_public_ip_on_launch = false

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-secondary-private-subnet"
  }
}

resource "aws_route_table_association" "secondary_private" {
  count          = var.secondary_availability_zone != null ? 1 : 0
  subnet_id      = aws_subnet.secondary_private[0].id
  route_table_id = aws_route_table.private.id
}

resource "aws_vpc_dhcp_options" "dhcp_options" {
  domain_name         = var.region == "us-east-1" ? "ec2.internal" : "${var.region}.compute.internal"
  domain_name_servers = ["AmazonProvidedDNS"]

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-dhcp-option-set"
  }
}

resource "aws_vpc_dhcp_options_association" "vpc_dhcp_options" {
  vpc_id          = local.vpc_id
  dhcp_options_id = aws_vpc_dhcp_options.dhcp_options.id
}

resource "aws_security_group" "public" {
  name        = "${var.project_name}-public-security-group"
  description = "Allow inbound connections from the VPC; allow connections on ports 22 (SSH) and 443 (HTTPS); allow all outbound connections"
  vpc_id      = local.vpc_id

  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = [local.vpc_cidr_block]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-public-security-group"
  }
}

resource "aws_security_group" "private" {
  name        = "${var.project_name}-private-security-group"
  description = "Allow all inbound and outbound connections within the VPC"
  vpc_id      = local.vpc_id

  ingress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = [local.vpc_cidr_block]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-private-security-group"
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

module "bastion" {
  source                = "../aws_host"
  availability_zone     = var.availability_zone
  project_name          = var.project_name
  name                  = "bastion"
  ami                   = var.bastion_host_ami
  instance_type         = var.bastion_host_instance_type
  ssh_key_name          = aws_key_pair.key_pair.key_name
  ssh_private_key_path  = var.ssh_private_key_path
  ssh_user              = var.ssh_user
  subnet_id             = aws_subnet.public.id
  vpc_security_group_id = aws_security_group.public.id
}
