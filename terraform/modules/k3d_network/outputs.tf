output "name" {
  value = docker_network.network.name
}

output "registry" {
  value = "${k3d_registry.docker_io_proxy.name}:5001"
}
