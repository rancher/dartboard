data "harvester_network" "this" {
  count = var.network_details.create_network ? 0 : 1
  name  = var.network_details.network_name
}
