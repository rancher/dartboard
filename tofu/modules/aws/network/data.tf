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

# Discover existing VPC subnets without case-sensitive tag filters, then match
# normalized tags in locals.
data "aws_subnets" "existing" {
  count = local.create_vpc ? 0 : 1

  filter {
    name   = "vpc-id"
    values = [one(data.aws_vpc.existing[*].id)]
  }
}

data "aws_subnet" "existing" {
  for_each = local.create_vpc ? toset([]) : toset(data.aws_subnets.existing[0].ids)

  id = each.value
}
