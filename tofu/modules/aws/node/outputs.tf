output "name" {
  value = var.name
}

output "private_name" {
  depends_on = [null_resource.host_configuration]
  value = var.public ? aws_instance.instance.public_dns : aws_instance.instance.private_dns
}

output "private_ip" {
  value = var.public ? aws_instance.instance.public_ip : aws_instance.instance.private_ip
}

output "public_ip" {
  value = var.public ? aws_instance.instance.public_ip : aws_instance.instance.private_ip
}

output "public_name" {
  depends_on = [null_resource.host_configuration]
  value      = var.public ? aws_instance.instance.public_ip : aws_instance.instance.private_dns
}

output "ssh_user" {
  value = var.ssh_user
}

output "ssh_key_path" {
  value = var.ssh_private_key_path
}
