output "private_name" {
  value = module.host.private_name
}

output "private_ip" {
  value = module.host.private_ip
}

output "public_name" {
  value = module.host.public_name
}

output "name" {
  value = var.name
}

output "ssh_script_filename" {
  value = local_file.ssh_script.filename
}
