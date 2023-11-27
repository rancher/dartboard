output "first_server_private_name" {
  value = var.server_count > 0 ? module.server_nodes[0].private_name : null
}

output "first_server_public_name" {
  value = var.server_count > 0 ? module.server_nodes[0].public_name : null
}

output "kubeconfig" {
  value = module.k3s.kubeconfig
}

output "context" {
  value = module.k3s.context
}

output "local_http_port" {
  value = var.local_http_port
}

output "local_https_port" {
  value = var.local_https_port
}

output "node_access_commands" {
  value = merge({
    for node in module.server_nodes : node.name => node.ssh_script_filename
    }, {
    for node in module.agent_nodes : node.name => node.ssh_script_filename
  })
}

output "ingress_class_name" {
  value = module.k3s.ingress_class_name
}
