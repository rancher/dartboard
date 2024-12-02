output "first_server_private_name" {
  value = var.server_count > 0 ? module.server_nodes[0].private_name : null
}

output "server_names" {
  value = [for node in module.server_nodes : node.private_name]
}
