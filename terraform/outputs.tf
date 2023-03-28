output "base_url" {
  value = "https://${local.upstream_san}:${local.rancher_port}"
}

output "bootstrap_password" {
  value = local.upstream_server_count > 0 ? module.rancher[0].bootstrap_password : null
}

output "downstream_cluster_names" {
  value = [local.downstream_san]
}

output "rancher_help" {
  value = <<-EOT
    UPSTREAM CLUSTER ACCESS:
      export KUBECONFIG=../config/upstream.yaml

    DOWNSTREAM CLUSTER ACCESS:
      export KUBECONFIG=../config/downstream.yaml

    RANCHER UI:
      https://${local.upstream_san}:${local.rancher_port}
 EOT
}
