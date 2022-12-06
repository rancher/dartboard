output "rancher_help" {
  value = <<-EOT
    CLUSTER ACCESS: already added to default kubeconfig

    RANCHER UI:
      https://${local.upstream_san}:3000

    MARIADB KINE DB:
      mariadb -h 127.0.0.1 -P 3306 -u kineuser --password=kinepassword kine

    POSTGRESQL KINE DB:
      PGPASSWORD=kinepassword psql -U kineuser -h localhost kine
 EOT
}
