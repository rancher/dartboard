output "rancher_help" {
  value = <<-EOT
    CLUSTER ACCESS: already added to default kubeconfig

    POSTGRESQL KINE DB:
      PGPASSWORD=kinepassword psql -U kineuser -h localhost kine
 EOT
}
