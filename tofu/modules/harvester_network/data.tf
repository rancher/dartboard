data "harvester_clusternetwork" "cluster-vlan" {
  count = var.create ? 0 : 1
  name  = var.clusternetwork_name
}

data "harvester_vlanconfig" "cluster-vlan-config" {
  count = var.create ? 0 : 1
  name  = var.vlanconfig_name
}

data "harvester_network" "this" {
  count = var.create ? 0 : 1
  name  = var.network_name
}
