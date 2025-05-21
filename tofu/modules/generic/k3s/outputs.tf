output "config" {
  value = {
    kubeconfig = abspath(local_file.kubeconfig.filename)
    context    = var.name

    // addresses of the Kubernetes API server
    kubernetes_addresses = {
      // resolvable over the Internet
      public = var.server_count > 0 ? "https://${module.server_nodes[0].public_name}:6443" : null
      // resolvable from the network this cluster runs in
      private = var.server_count > 0 ? "https://${module.server_nodes[0].private_name}:6443" : null
      // resolvable from the host running OpenTofu when create_tunnels == true
      tunnel = local.local_kubernetes_api_url
    }

    // addresses of applications running in this cluster
    app_addresses = {
      public = { // resolvable over the Internet
        name       = var.server_count > 0 ? module.server_nodes[0].public_name : null
        http_port  = 80
        https_port = 443
      }
      private = { // resolvable from the network this cluster runs in
        name       = var.server_count > 0 ? module.server_nodes[0].private_name : null
        http_port  = 80
        https_port = 443
      }
      tunnel = { // resolvable from the host running OpenTofu when create_tunnels == true
        name       = "${var.name}.local.gd"
        http_port  = var.tunnel_app_http_port
        https_port = var.tunnel_app_https_port
      }
    }

    node_access_commands = merge({
      for node in module.server_nodes : node.name => abspath(node.ssh_script_filename)
      }, {
      for node in module.agent_nodes : node.name => abspath(node.ssh_script_filename)
    })
    ingress_class_name          = null
    reserve_node_for_monitoring = var.reserve_node_for_monitoring
    server_nodes = module.server_nodes
    agent_nodes = module.agent_nodes
  }
}
