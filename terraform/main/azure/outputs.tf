output "clusters" {
  value = {
    for i, cluster in local.clusters : cluster.name => {
      local_name           = cluster.local_name,
      local_http_port      = module.cluster[i].local_http_port
      local_https_port     = module.cluster[i].local_https_port
      private_name         = module.cluster[i].first_server_private_name
      kubeconfig           = module.cluster[i].kubeconfig
      context              = module.cluster[i].context
      node_access_commands = module.cluster[i].node_access_commands
    }
  }
}
