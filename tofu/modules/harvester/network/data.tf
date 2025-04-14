data "harvester_network" "this" {
  count     = var.network_details.create ? 0 : 1
  name      = var.network_details.name
  namespace = var.network_details.namespace
}
