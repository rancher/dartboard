output "clusters" {
  value = {
    for i, cluster in local.clusters : cluster.name => {
      kubeconfig = module.cluster[i].kubeconfig
      context    = module.cluster[i].context

      // alternative URL to reach the API from the same network this cluster is in
      private_kubernetes_api_url = "https://${module.cluster[i].first_server_private_name}:6443"

      // addresses of applications running in this cluster
      app_addresses = {
        public = { // resolvable over the Internet
          name       = module.cluster[i].first_server_private_name
          http_port  = 80
          https_port = 443
        }
        private = {         // resolvable from the network this cluster runs in
          name       = null // only public supported
          http_port  = null
          https_port = null
        }
        tunnel = {          // resolvable from the host running Terraform
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
