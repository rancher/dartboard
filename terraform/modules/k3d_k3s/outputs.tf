output "kubeconfig" {
  value = pathexpand("~/.kube/config")
}

output "context" {
  value = "k3d-${var.project_name}-${var.name}"
}
