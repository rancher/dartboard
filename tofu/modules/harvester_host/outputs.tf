output "id" {
  value = harvester_virtualmachine.this.id
}

output "private_network_interfaces" {
  value = local.private_network_interfaces
}

output "private_address" {
  value = length(local.private_network_interfaces) > 0 ? local.private_network_interfaces[0].ip_address : null
}

output "public_network_interfaces" {
  value = local.public_network_interfaces
}

output "public_address" {
  value = length(local.public_network_interfaces) > 0 ? local.public_network_interfaces[0].ip_address : null
}

output "name" {
  value = var.name
}

output "ssh_script_filename" {
  value = abspath(module.ssh_access.ssh_script_filename)
}

output "cloudinit_splat" {
  value = data.harvester_cloudinit_secret.this[*]
}
