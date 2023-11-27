output "kubeconfig" {
  value = abspath(local_file.kubeconfig.filename)
}

output "context" {
  value = "${var.project_name}-${var.name}"
}

output "cluster_public_name" {
  value = azurerm_kubernetes_cluster.cluster.fqdn
}

output "node_access_commands" {
  value = []
}

output "ingress_class_name" {
  value = "webapprouting.kubernetes.azure.com"
}
