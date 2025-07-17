# VPC resource created only when existing_vpc_name is null
resource "aws_vpc" "main" {
  count                = local.create_vpc ? 1 : 0
  cidr_block           = "172.16.0.0/16"
  enable_dns_support   = true
  enable_dns_hostnames = true

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-vpc"
  }
}

# Update locals to use coalescing for resource selection
locals {
  vpc_id         = coalesce(one(aws_vpc.main[*].id), one(data.aws_vpc.existing[*].id))
  vpc_cidr_block = coalesce(one(aws_vpc.main[*].cidr_block), one(data.aws_vpc.existing[*].cidr_block))
  internet_gateway_id = coalesce(one(aws_internet_gateway.main[*].id), one(data.aws_internet_gateway.existing[*].id))

  public_subnet_id = coalesce(one(aws_subnet.public[*].id), one(data.aws_subnet.public[*].id))
  private_subnet_id = coalesce(one(aws_subnet.private[*].id), one(data.aws_subnet.private[*].id))
  secondary_private_subnet_id = (local.create_vpc && var.secondary_availability_zone != null) ? aws_subnet.secondary_private[0].id : (!local.create_vpc && var.secondary_availability_zone != null) ? data.aws_subnet.secondary_private[0].id : null

  create_vpc = var.existing_vpc_name == null
}

resource "aws_internet_gateway" "main" {
  count  = local.create_vpc ? 1 : 0
  vpc_id = local.vpc_id

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-internet-gateway"
  }
}

resource "aws_eip" "nat_eip" {

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-nat-eip"
  }
}

resource "aws_nat_gateway" "nat" {
  allocation_id = aws_eip.nat_eip.id
  subnet_id     = local.public_subnet_id

  depends_on = [data.aws_internet_gateway.existing, aws_internet_gateway.main]

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-nat"
  }
}

resource "aws_subnet" "public" {
  count                   = local.create_vpc ? 1 : 0
  availability_zone       = var.availability_zone
  vpc_id                  = local.vpc_id
  cidr_block              = "172.16.0.0/20"
  map_public_ip_on_launch = true

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-public-subnet"
  }
}

resource "aws_subnet" "private" {
  count                   = local.create_vpc ? 1 : 0
  availability_zone       = var.availability_zone
  vpc_id                  = local.vpc_id
  cidr_block              = "172.16.96.0/20"
  map_public_ip_on_launch = false

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-private-subnet"
  }
}

resource "aws_subnet" "secondary_private" {
  count                   = local.create_vpc && var.secondary_availability_zone != null ? 1 : 0
  availability_zone       = var.secondary_availability_zone
  vpc_id                  = local.vpc_id
  cidr_block              = "172.16.192.0/20"
  map_public_ip_on_launch = false

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-secondary-private-subnet"
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

resource "aws_main_route_table_association" "vpc_internet" {
  vpc_id         = local.vpc_id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table" "public" {
  vpc_id = local.vpc_id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = local.internet_gateway_id
  }

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-public-route-table"
  }
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

resource "aws_route_table_association" "public" {
  subnet_id      = local.public_subnet_id
  route_table_id = aws_route_table.public.id
}

resource "aws_route_table_association" "private" {
  subnet_id      = local.private_subnet_id
  route_table_id = aws_route_table.private.id
}

resource "aws_route_table_association" "secondary_private" {
  count          = local.create_vpc && var.secondary_availability_zone != null ? 1 : 0
  subnet_id      = local.secondary_private_subnet_id
  route_table_id = aws_route_table.private.id
}

resource "aws_vpc_dhcp_options" "dhcp_options" {
  count               = local.create_vpc ? 1 : 0
  domain_name         = var.region == "us-east-1" ? "ec2.internal" : "${var.region}.compute.internal"
  domain_name_servers = ["AmazonProvidedDNS"]

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-dhcp-option-set"
  }
}

resource "aws_vpc_dhcp_options_association" "vpc_dhcp_options" {
  count           = local.create_vpc ? 1 : 0
  vpc_id          = local.vpc_id
  dhcp_options_id = aws_vpc_dhcp_options.dhcp_options[0].id
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

# Update the bastion module configuration
module "bastion" {
  source               = "../node"
  project_name         = var.project_name
  name                 = "bastion"
  ssh_private_key_path = var.ssh_private_key_path
  ssh_user             = var.ssh_bastion_user
  public               = true
  node_module_variables = {
    ami : var.bastion_host_ami
    instance_type : var.bastion_host_instance_type
    root_volume_size_gb : 30
    host_configuration_commands : []
  }
  network_config = {
    availability_zone : var.availability_zone,
    public_subnet_id : local.public_subnet_id
    private_subnet_id : local.private_subnet_id
    secondary_private_subnet_id : local.secondary_private_subnet_id
    public_security_group_id : aws_security_group.public.id
    private_security_group_id : aws_security_group.private.id
    ssh_key_name : aws_key_pair.key_pair.key_name
    ssh_bastion_host : null,
    ssh_bastion_user : null,
  }
}
