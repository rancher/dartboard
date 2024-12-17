output "config" {
  value = {
    kubeconfig = abspath(local_file.kubeconfig.filename)
    context    = var.name

    // addresses of the Kubernetes API server
    kubernetes_addresses = {
      // resolvable over the Internet
      public = "https://${azurerm_kubernetes_cluster.cluster.fqdn}:443"
      // resolvable from the network this cluster runs in
      private = "https://${azurerm_kubernetes_cluster.cluster.fqdn}:443"
      // resolvable from the host running OpenTofu
      tunnel = "https://${azurerm_kubernetes_cluster.cluster.fqdn}:443"
    }

    // addresses of applications running in this cluster
    app_addresses = {
      public = { // resolvable over the Internet
        name       = null
        http_port  = null
        https_port = null
      }
      private = { // resolvable from the network this cluster runs in
        name       = null
        http_port  = null
        https_port = null
      }
      tunnel = { // resolvable from the host running OpenTofu
        name       = null
        http_port  = null
        https_port = null
      }
    }

    node_access_commands        = {}
    ingress_class_name          = "webapprouting.kubernetes.azure.com"
    reserve_node_for_monitoring = var.reserve_node_for_monitoring
  }
}
