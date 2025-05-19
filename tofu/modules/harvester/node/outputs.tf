output "name" {
  value = var.name
}

output "id" {
  value = harvester_virtualmachine.this.id
}

output "private_name" {
  value = "${var.network_config.public ? local.public_network_interfaces[0].ip_address : local.private_network_interfaces[0].ip_address}.sslip.io"
}

output "private_ip" {
  value = var.network_config.public ? local.public_network_interfaces[0].ip_address : local.private_network_interfaces[0].ip_address
}

output "public_ip" {
  value = var.network_config.public ? local.public_network_interfaces[0].ip_address : local.private_network_interfaces[0].ip_address
}

output "public_name" {
  value = "${var.network_config.public ? local.public_network_interfaces[0].ip_address : local.private_network_interfaces[0].ip_address}.sslip.io"
}

output "ssh_user" {
  value = var.ssh_user
}

output "ssh_key_path" {
  value = var.ssh_private_key_path
}
