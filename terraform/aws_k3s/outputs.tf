output "first_server_private_name" {
  value = var.server_count > 0 ? module.server_nodes[0].private_name : null
}

output "kubeconfig" {
  value = module.k3s.kubeconfig
}

output "context" {
  value = module.k3s.context
}

output "ssh_scripts" {
  value = merge({
  for node in module.server_nodes : node.name => { ssh_script : node.ssh_script_filename }
  }, {
  for node in module.agent_nodes : node.name => { ssh_script : node.ssh_script_filename }
  })
}
