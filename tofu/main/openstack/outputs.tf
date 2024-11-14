output "clusters" {
  value = {
    for i, cluster in local.clusters : cluster.name => {
      kubeconfig = module.cluster[i].kubeconfig
      context    = module.cluster[i].context

      // addresses of the Kubernetes API server
      kubernetes_addresses = {
        // resolvable over the Internet
        public = "https://${module.cluster[i].first_server_public_name}:6443"
        // resolvable from the network this cluster runs in
        private = "https://${module.cluster[i].first_server_private_name}:6443"
        // resolvable from the host running OpenTofu
        tunnel = module.cluster[i].local_kubernetes_api_url
      }

      // addresses of applications running in this cluster
      app_addresses = {
        public = { // resolvable over the Internet
          name       = module.cluster[i].public_dns
          http_port  = 80
          https_port = 443
        }
        private = {         // resolvable from the network this cluster runs in
          name       = null // only public supported
          http_port  = null
          https_port = null
        }
        tunnel = {          // resolvable from the host running OpenTofu
          name       = null // tunnels not supported
          http_port  = null
          https_port = null
        }
      }

      node_access_commands = module.cluster[i].node_access_commands
      ingress_class_name   = module.cluster[i].ingress_class_name
    }
  }
}
