output "old_upstream_san" {
  value = local.old_upstream_san
}

output "old_upstream_public_port" {
  value = local.old_upstream_public_port
}

output "old_upstream_cluster_private_name" {
  value = module.old_upstream_cluster.first_server_private_name
}

output "old_upstream_cluster" {
  value = { kubeconfig : pathexpand("~/.kube/config"), context : "k3d-${local.project_name}-oldupstream" }
}

output "old_downstream_clusters" {
  value = [
    {
      name : "olddownstream", kubeconfig : pathexpand("~/.kube/config"),
      context : "k3d-${local.project_name}-olddownstream"
    }
  ]
}

output "new_upstream_san" {
  value = local.new_upstream_san
}

output "new_upstream_public_port" {
  value = local.new_upstream_public_port
}

output "new_upstream_cluster_private_name" {
  value = module.new_upstream_cluster.first_server_private_name
}

output "new_upstream_cluster" {
  value = { kubeconfig : pathexpand("~/.kube/config"), context : "k3d-${local.project_name}-newupstream" }
}

output "new_downstream_clusters" {
  value = [
    {
      name : "newdownstream", kubeconfig : pathexpand("~/.kube/config"),
      context : "k3d-${local.project_name}-newdownstream"
    }
  ]
}
