// connect with
// PGPASSWORD=kinepassword psql -U kineuser -h localhost kine

output "private_name" {
  value = module.server_node.private_name
}
