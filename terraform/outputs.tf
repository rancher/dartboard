output "rancher_help" {
  value = <<-EOT
    CLUSTER ACCESS: already added to default kubeconfig

    RANCHER UI:
      https://upstream.local.gd:8443

    CLUSTER API (upstream):
      https://upstream.local.gd:6445

    CLUSTER API (downstream):
      https://downstream.local.gd:6446
 EOT
}
