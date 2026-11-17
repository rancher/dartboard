output "config" {
  # Fail the plan with a clear message when an existing VPC lacks a matching subnet
  precondition {
    condition     = local.create_vpc || local.existing_public_subnet_id != null
    error_message = "No public subnet found in VPC '${coalesce(var.existing_vpc_name, "<unset>")}', AZ '${var.availability_zone}': need a Name tag containing \"public\" or tag Tier = Public. Subnets tagged as both public and private are ignored."
  }
  precondition {
    condition     = local.create_vpc || local.existing_private_subnet_id != null
    error_message = "No private subnet found in VPC '${coalesce(var.existing_vpc_name, "<unset>")}', AZ '${var.availability_zone}': need a Name tag containing \"private\" or tag Tier = Private. Subnets tagged as both public and private, or as secondary, are ignored."
  }
  precondition {
    condition     = local.create_vpc || var.secondary_availability_zone == null || local.existing_secondary_private_subnet_id != null
    error_message = "No secondary private subnet found in VPC '${coalesce(var.existing_vpc_name, "<unset>")}', AZ '${coalesce(var.secondary_availability_zone, "<unset>")}': need a Name tag containing \"secondary\" and \"private\" or tag Tier = SecondaryPrivate. Subnets tagged as both public and private are ignored."
  }

  value = {
    availability_zone : var.availability_zone,
    public_subnet_id : local.public_subnet_id,
    private_subnet_id : local.private_subnet_id,
    secondary_private_subnet_id : local.secondary_private_subnet_id,
    public_security_group_id : aws_security_group.public.id,
    private_security_group_id : aws_security_group.private.id,
    other_security_group_ids : [aws_security_group.ssh_ipv4.id],
    ssh_key_name : aws_key_pair.key_pair.key_name,
    ssh_bastion_host : module.bastion.public_name,
    ssh_bastion_user : var.ssh_bastion_user,
    ssh_bastion_key_path : var.ssh_private_key_path,
  }
}
