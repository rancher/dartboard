output "name" {
  value = var.name
}

output "private_name" {
  depends_on = [null_resource.host_configuration]
  value = aws_instance.instance.private_dns
}

output "private_ip" {
  value = aws_instance.instance.private_ip
}

output "public_name" {
  depends_on = [null_resource.host_configuration]
  value      = aws_instance.instance.public_dns
}

output "public_ip" {
  value = aws_instance.instance.public_ip
}

output "ssh_user" {
  value = var.ssh_user
}

output "ssh_key_path" {
  value = var.ssh_private_key_path
}
