locals {
  k3s_outputs = { for i, cluster in local.k3s_clusters : cluster.name => {
    kubeconfig = module.k3s_cluster[i].kubeconfig
    context    = module.k3s_cluster[i].context

    // addresses of the Kubernetes API server
    kubernetes_addresses = {
      // resolvable over the Internet
      public = "https://${module.k3s_cluster[i].first_server_public_name}:6443"
      // resolvable from the network this cluster runs in
      private = "https://${module.k3s_cluster[i].first_server_private_name}:6443"
      // resolvable from the host running OpenTofu
      tunnel = module.k3s_cluster[i].local_kubernetes_api_url
    }

    // addresses of applications running in this cluster
    app_addresses = {
      public = { // resolvable over the Internet
        name       = module.k3s_cluster[i].first_server_public_name
        http_port  = 80
        https_port = 443
      }
      private = { // resolvable from the network this cluster runs in
        name       = module.k3s_cluster[i].first_server_private_name
        http_port  = 80
        https_port = 443
      }
      tunnel = { // resolvable from the host running OpenTofu
        name       = "${cluster.name}.local.gd"
        http_port  = module.k3s_cluster[i].tunnel_app_http_port
        https_port = module.k3s_cluster[i].tunnel_app_https_port
      }
    }

    node_access_commands = module.k3s_cluster[i].node_access_commands
    ingress_class_name   = module.k3s_cluster[i].ingress_class_name
    }
  }
  rke2_outputs = { for i, cluster in local.rke2_clusters : cluster.name => {
    kubeconfig = module.rke2_cluster[i].kubeconfig
    context    = module.rke2_cluster[i].context

    // addresses of the Kubernetes API server
    kubernetes_addresses = {
      // resolvable over the Internet
      public = "https://${module.rke2_cluster[i].first_server_public_name}:6443"
      // resolvable from the network this cluster runs in
      private = "https://${module.rke2_cluster[i].first_server_private_name}:6443"
      // resolvable from the host running OpenTofu
      tunnel = module.rke2_cluster[i].local_kubernetes_api_url
    }

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
        name       = "${cluster.name}.local.gd"
        http_port  = module.rke2_cluster[i].tunnel_app_http_port
        https_port = module.rke2_cluster[i].tunnel_app_https_port
      }
    }

    node_access_commands = module.rke2_cluster[i].node_access_commands
    ingress_class_name   = module.rke2_cluster[i].ingress_class_name
    }
  }
  aks_outputs = { for i, cluster in local.aks_clusters : cluster.name => {
    kubeconfig = module.aks_cluster[i].kubeconfig
    context    = module.aks_cluster[i].context

    // addresses of the Kubernetes API server
    kubernetes_addresses = {
      // resolvable over the Internet
      public = "https://${module.aks_cluster[i].cluster_public_name}:443"
      // resolvable from the network this cluster runs in
      private = "https://${module.aks_cluster[i].cluster_public_name}:443"
      // resolvable from the host running OpenTofu
      tunnel = "https://${module.aks_cluster[i].cluster_public_name}:443"
    }

    // addresses of applications running in this cluster
    app_addresses = {
      public = {          // resolvable over the Internet
        name       = null // not known at the OpenTofu stage, will depend on LoadBalancers in Kubernetes
        http_port  = null
        https_port = null
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

    node_access_commands = module.aks_cluster[i].node_access_commands
    ingress_class_name   = module.aks_cluster[i].ingress_class_name
    }
  }
}

output "clusters" {
  value = merge(local.k3s_outputs, local.rke2_outputs, local.aks_outputs)
}
