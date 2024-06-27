output "name" {
  value = docker_network.network.name
}

output "pull_proxy_registries" {
  value = [
    for i in range(length(var.registry_pull_proxies)) :
    {
      name    = var.registry_pull_proxies[i].name
      address = "k3d-${k3d_registry.proxy[i].name}:5000"
    }
  ]
}
