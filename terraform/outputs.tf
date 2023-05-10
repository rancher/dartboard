output "upstream_san" {
  value = local.upstream_san
}

output "upstream_public_port" {
  value = local.upstream_public_port
}

output "upstream_cluster_private_name" {
  value = module.upstream_cluster.first_server_private_name
}

output "upstream_cluster" {
  value = { kubeconfig : pathexpand("~/.kube/config"), context : "k3d-${local.project_name}-upstream" }
}

output "downstream_clusters" {
  value = [
    { name : "downstream", kubeconfig : pathexpand("~/.kube/config"), context : "k3d-${local.project_name}-downstream" }
  ]
}
