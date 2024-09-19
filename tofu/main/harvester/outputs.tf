locals {
  # k3s_outputs = { for i, cluster in local.k3s_clusters : cluster.name_prefix => {
  #   kubeconfig = module.k3s_cluster[i].kubeconfig
  #   context    = module.k3s_cluster[i].context

  #   // alternative URL to reach the API from the same network this cluster is in
  #   private_kubernetes_api_url = "https://${module.k3s_cluster[i].first_server_private_name}:6443"
  #   master_url = "https://${cluster.name_prefix}.local.gd:${var.first_kubernetes_api_port}"

  #   // addresses of applications running in this cluster
  #   app_addresses = {
  #     public = { // resolvable over the Internet
  #       name       = module.k3s_cluster[i].first_server_public_name
  #       http_port  = 80
  #       https_port = 443
  #     }
  #     private = { // resolvable from the network this cluster runs in
  #       name       = module.k3s_cluster[i].first_server_private_name
  #       http_port  = 80
  #       https_port = 443
  #     }
  #     tunnel = { // resolvable from the host running OpenTofu
  #       name       = "${cluster.name_prefix}.local.gd"
  #       http_port  = module.k3s_cluster[i].tunnel_app_http_port
  #       https_port = module.k3s_cluster[i].tunnel_app_https_port
  #     }
  #   }

    # node_access_commands = module.k3s_cluster[i].node_access_commands
    # ingress_class_name   = module.k3s_cluster[i].ingress_class_name
    # }
  # }
  rke2_outputs = { for i, cluster in local.rke2_clusters : cluster.name_prefix => {
    kubeconfig = module.rke2_cluster[i].kubeconfig
    context    = module.rke2_cluster[i].context

    // alternative URL to reach the API from the same network this cluster is in
    private_kubernetes_api_url = "https://${module.rke2_cluster[i].first_server_private_name}:6443"
    master_url = "https://${cluster.name_prefix}.local.gd:${var.first_kubernetes_api_port}"

    // addresses of applications running in this cluster
    app_addresses = {
      public = { // resolvable over the Internet
        name       = module.rke2_cluster[i].first_server_public_name
        http_port  = 80
        https_port = 443
      }
      private = { // resolvable from the network this cluster runs in
        name       = module.rke2_cluster[i].first_server_private_name
        http_port  = 80
        https_port = 443
      }
      tunnel = { // resolvable from the host running OpenTofu
        name       = "${cluster.name_prefix}.local.gd"
        http_port  = module.rke2_cluster[i].tunnel_app_http_port
        https_port = module.rke2_cluster[i].tunnel_app_https_port
      }
    }

    node_access_commands = module.rke2_cluster[i].node_access_commands
    ingress_class_name   = module.rke2_cluster[i].ingress_class_name
    }
  }
}

output "clusters" {
  # value = merge(local.k3s_outputs, local.rke2_outputs)
    value = local.rke2_outputs
}
