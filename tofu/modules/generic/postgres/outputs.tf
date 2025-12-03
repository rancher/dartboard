// connect with
// PGPASSWORD=kinepassword psql -U kineuser -h localhost kine

output "datastore_endpoint" {
  value = "postgres://kineuser:${var.kine_password}@${module.server_node.private_name}/kine"
}
