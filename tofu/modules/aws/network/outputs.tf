output "config" {
  value = {
    availability_zone : var.availability_zone,
    public_subnet_id : local.public_subnet_id,
    private_subnet_id : local.private_subnet_id,
    secondary_private_subnet_id : var.secondary_availability_zone != null ? local.secondary_private_subnet_id : null,
    public_security_group_id : aws_security_group.public.id,
    private_security_group_id : aws_security_group.private.id,
    ssh_key_name : aws_key_pair.key_pair.key_name,
    ssh_bastion_host : module.bastion.public_name,
    ssh_bastion_user : var.ssh_bastion_user,
  }
}
