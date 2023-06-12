output "public_subnet_id" {
  value = aws_subnet.public.id
}

output "private_subnet_id" {
  value = aws_subnet.private.id
}

output "secondary_private_subnet_id" {
  value = var.secondary_availability_zone != null ? aws_subnet.secondary_private[0].id : null
}

output "public_security_group_id" {
  value = aws_security_group.public.id
}

output "private_security_group_id" {
  value = aws_security_group.private.id
}

output "key_name" {
  value = aws_key_pair.key_pair.key_name
}
