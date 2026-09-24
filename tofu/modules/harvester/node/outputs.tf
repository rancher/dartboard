output "name" {
  value = var.name
}

output "id" {
  value = harvester_virtualmachine.this.id
}

output "private_name" {
  depends_on = [null_resource.host_configuration]
  value      = "${local.node_ip}.sslip.io"
}

output "private_ip" {
  depends_on = [null_resource.host_configuration]
  value      = local.node_ip
}

output "public_name" {
  depends_on = [null_resource.host_configuration]
  value      = "${local.node_ip}.sslip.io"
}

output "public_ip" {
  depends_on = [null_resource.host_configuration]
  value      = local.node_ip
}

output "ssh_user" {
  value = var.ssh_user
}

output "ssh_key_path" {
  value = var.ssh_private_key_path
}
