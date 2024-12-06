output "clusters" {
  value = {
    for i, cluster in local.all_clusters : cluster.name => {
      kubeconfig = module.cluster[i].kubeconfig
      context    = module.cluster[i].context

      // addresses of the Kubernetes API server
      kubernetes_addresses = {
        // resolvable over the Internet
        public = null
        // resolvable from the network this cluster runs in
        private = "https://${module.cluster[i].first_server_private_name}:6443"
        // resolvable from the host running OpenTofu
        tunnel = module.cluster[i].local_kubernetes_api_url
      }

      // addresses of applications running in this cluster
      app_addresses = {
        public = { // resolvable over the Internet
          name       = null
          http_port  = null
          https_port = null
        }
        private = { // resolvable from the network this cluster runs in
          name       = module.cluster[i].first_server_private_name
          http_port  = 80
          https_port = 443
        }
        tunnel = { // resolvable from the host running OpenTofu
          name       = "${cluster.name}.local.gd"
          http_port  = module.cluster[i].app_http_port
          https_port = module.cluster[i].app_https_port
        }
      }

      node_access_commands = {}
      ingress_class_name   = null
      reserve_node_for_monitoring = cluster.reserve_node_for_monitoring
    }
  }
}
