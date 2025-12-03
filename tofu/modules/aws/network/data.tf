# Data source to look up existing VPC
data "aws_vpc" "existing" {
  count = local.create_vpc ? 0 : 1

  filter {
    name   = "tag:Name"
    values = [var.existing_vpc_name]
  }
}

data "aws_internet_gateway" "existing" {
  count = local.create_vpc ? 0 : 1
  filter {
    name   = "attachment.vpc-id"
    values = [local.vpc_id]
  }
}

# Data sources to look up existing subnets
data "aws_subnet" "public" {
  count             = local.create_vpc ? 0 : 1
  vpc_id            = one(data.aws_vpc.existing[*].id)
  availability_zone = var.availability_zone

  tags = {
    Name = "*public*",
    Tier = "Public"
  }
}

data "aws_subnet" "private" {
  count             = !local.create_vpc ? 1 : 0
  vpc_id            = one(data.aws_vpc.existing[*].id)
  availability_zone = var.availability_zone

  tags = {
    Name = "*private*"
    Tier = "Private"
  }
}

data "aws_subnet" "secondary_private" {
  count             = !local.create_vpc && var.secondary_availability_zone != null ? 1 : 0
  vpc_id            = one(data.aws_vpc.existing[*].id)
  availability_zone = var.secondary_availability_zone

  tags = {
    Name = "*secondary*private*"
    Tier = "SecondaryPrivate"
  }
}
