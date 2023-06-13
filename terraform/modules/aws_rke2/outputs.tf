output "first_server_private_name" {
  value = var.server_count > 0 ? module.server_nodes[0].private_name : null
}

output "kubeconfig" {
  value = module.rke2.kubeconfig
}

output "context" {
  value = module.rke2.context
}

output "local_http_port" {
  value = var.local_http_port
}

output "local_https_port" {
  value = var.local_https_port
}

output "ssh_scripts" {
  value = merge({
  for node in module.server_nodes : node.name => { ssh_script : node.ssh_script_filename }
  }, {
  for node in module.agent_nodes : node.name => { ssh_script : node.ssh_script_filename }
  })
}
