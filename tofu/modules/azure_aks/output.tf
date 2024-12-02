// note: hosts in this file need to be resolvable from the host running OpenTofu
output "kubeconfig" {
  value = abspath(local_file.kubeconfig.filename)
}

// note: must match the host in kubeconfig
output "local_kubernetes_api_url" {
  value = "https://${azurerm_kubernetes_cluster.cluster.kube_config.host}:6443"
}

output "context" {
  value = "${var.project_name}-${var.name}"
}

output "cluster_public_name" {
  value = azurerm_kubernetes_cluster.cluster.fqdn
}

output "node_access_commands" {
  value = {}
}

output "ingress_class_name" {
  value = "webapprouting.kubernetes.azure.com"
}
