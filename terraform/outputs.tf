output "clusters" {
  value = {
    for name, locals in local.clusters : name => {
      name : name,
      san : locals.san,
      public_http_port : locals.public_http_port,
      public_https_port : locals.public_https_port,
      private_name        = module.cluster[name].first_server_private_name,
      kubernetes_api_port = locals.kubernetes_api_port
      kubeconfig          = pathexpand("~/.kube/config")
      context             = "k3d-${local.project_name}-${name}"
    }
  }
}
