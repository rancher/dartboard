output "config" {
  value = {
    availability_zone : var.availability_zone,
    public_subnet_id : local.public_subnet_id,
    private_subnet_id : local.private_subnet_id,
    secondary_private_subnet_id : local.secondary_private_subnet_id,
    public_security_group_id : aws_security_group.public.id,
    private_security_group_id : aws_security_group.private.id,
    other_security_group_ids: [aws_security_group.ssh_ipv4.id],
    ssh_key_name : aws_key_pair.key_pair.key_name,
    ssh_bastion_host : module.bastion.public_name,
    ssh_bastion_user : var.ssh_bastion_user,
  }
}
