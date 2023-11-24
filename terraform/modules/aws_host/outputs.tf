output "id" {
  value = aws_instance.instance.id
}

output "private_name" {
  value = aws_instance.instance.private_dns
}

output "private_ip" {
  value = aws_instance.instance.private_ip
}

output "public_name" {
  depends_on = [null_resource.host_configuration]
  value      = aws_instance.instance.public_dns
}

output "name" {
  value = var.name
}

output "ssh_script_filename" {
  value = abspath(module.ssh_access.ssh_script_filename)
}
