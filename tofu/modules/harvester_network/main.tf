resource "harvester_ssh_key" "public_key" {
  name      = "${var.network_name}-key"
  namespace = var.namespace

  public_key = file(var.ssh_public_key_path)
}

resource "harvester_clusternetwork" "cluster-vlan" {
  count       = var.create ? 1 : 0
  name        = var.clusternetwork_name
  description = "Cluster VLAN managed by Dartboard's Harvester opentofu module"
}

resource "harvester_vlanconfig" "cluster-vlan-config" {
  count = var.create ? 1 : 0
  name  =  var.vlanconfig_name

  cluster_network_name = harvester_clusternetwork.cluster-vlan[0].name

  uplink {
    nics        = var.vlan_uplink.nics
    bond_mode   = var.vlan_uplink.bond_mode
    bond_miimon = var.vlan_uplink.bond_miimon
    mtu         = var.vlan_uplink.mtu
  }
}

resource "harvester_network" "this" {
  count       = var.create ? 1 : 0
  depends_on  = [ harvester_vlanconfig.cluster-vlan-config ]
  name        = var.network_name
  namespace   = var.namespace
  description = "Harvester network managed by Dartboard's Harvester opentofu module"

  vlan_id = var.vlan_id

  route_mode            = var.route_mode
  route_dhcp_server_ip  = var.route_dhcp_server_ip
  route_cidr            = var.route_cidr
  route_gateway         = var.route_gateway

  cluster_network_name = data.harvester_clusternetwork.cluster-vlan[0].name
}
