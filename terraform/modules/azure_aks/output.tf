output "first_server_private_name" {
  value = azurerm_kubernetes_cluster.cluster.private_fqdn
}

output "first_server_public_name" {
  value = azurerm_kubernetes_cluster.cluster.fqdn
}

output "kubeconfig" {
  value = abspath(local_file.kubeconfig.filename)
}

output "context" {
  value = "${var.project_name}-${var.name}"
}

output "local_http_port" {
  value = var.local_http_port
}

output "local_https_port" {
  value = var.local_https_port
}

output "node_access_commands" {
  value = []
}
