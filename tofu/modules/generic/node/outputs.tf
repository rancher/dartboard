output "name" {
  value = var.name
}

output "public_ip" {
  value = module.host.public_ip
}

output "public_name" {
  value = module.host.public_name
}

output "private_ip" {
  value = module.host.private_ip
}

output "private_name" {
  value = module.host.private_name
}

output "ssh_user" {
  value = module.host.ssh_user
}

output "ssh_key_path" {
  value = module.host.ssh_key_path
}

output "ssh_script_filename" {
  value = local_file.ssh_script.filename
}
