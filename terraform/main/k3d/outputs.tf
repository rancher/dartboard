output "clusters" {
  value = {
  for name, locals in local.clusters : name => {
    name : name,
    san : locals.san,
    public_http_port : locals.public_http_port,
    public_https_port : locals.public_https_port,
    private_name = module.cluster[name].first_server_private_name,
    kubeconfig   = module.cluster[name].kubeconfig
    context      = module.cluster[name].context
  }
  }
}
