// connect with
// PGPASSWORD=kinepassword psql -U kineuser -h localhost kine

output "datastore_endpoint" {
  value = "postgres://kineuser:kinepassword@${module.server_node.private_name}/kine"
}
