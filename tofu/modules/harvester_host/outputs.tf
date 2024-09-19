output "id" {
  value = harvester_virtualmachine.this.id
}

output "private_network_interfaces" {
  value = local.private_network_interfaces
}

output "public_network_interfaces" {
  value = local.public_network_interfaces
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
