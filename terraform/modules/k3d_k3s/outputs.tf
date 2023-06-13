output "first_server_private_name" {
  value = "k3d-${var.project_name}-${var.name}-server-0"
}

output "kubeconfig" {
  value = pathexpand("~/.kube/config")
}

output "context" {
  value = "k3d-${var.project_name}-${var.name}"
}

output "local_http_port" {
  value = var.local_http_port
}

output "local_https_port" {
  value = var.local_https_port
}
