// connect with
//

output "datastore_endpoint" {
  value = join(",", formatlist("http://%s:2379", module.server_nodes.*.private_ip))
}
