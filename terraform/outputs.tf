output "base_url" {
  value = "https://${local.upstream_san}:${local.rancher_port}"
}

output "bootstrap_password" {
  value = local.upstream_server_count > 0 ? module.rancher[0].bootstrap_password : null
}

output "upstream_cluster" {
  value = {kubeconfig:pathexpand("../config/upstream.yaml"), context:"upstream.local.gd"}
}

output "downstream_clusters" {
  value = [{name: "downstream", kubeconfig:pathexpand("../config/downstream.yaml"), context:"downstream.local.gd"}]
}
