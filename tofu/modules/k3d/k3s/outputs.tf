output "config" {
  value = {
    kubeconfig = var.server_count > 0 ? abspath(local_file.kubeconfig[0].filename) : null
    context    = var.name
    name       = local.k3d_cluster_name

    // addresses of the Kubernetes API server
    kubernetes_addresses = {
      // resolvable over the Internet
      public = null
      // resolvable from the network this cluster runs in
      private = "k3d-${var.project_name}-${var.name}-server-0"
      // resolvable from the host running OpenTofu
      tunnel = local.local_kubernetes_api_url
    }

    // addresses of applications running in this cluster
    app_addresses = {
      public = { // resolvable over the Internet
        name       = null
        http_port  = 80
        https_port = 443
      }
      private = { // resolvable from the network this cluster runs in
        name       = "k3d-${var.project_name}-${var.name}-server-0"
        http_port  = 80
        https_port = 443
      }
      tunnel = { // resolvable from the host running OpenTofu
        name       = "${var.name}.local.gd"
        http_port  = var.tunnel_app_http_port
        https_port = var.tunnel_app_https_port
      }
    }

    node_access_commands        = {}
    ingress_class_name          = null
    reserve_node_for_monitoring = var.reserve_node_for_monitoring
  }
}
