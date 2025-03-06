terraform {
  required_providers {
    harvester = {
      source = "harvester/harvester"
    }
  }
}

resource "harvester_ssh_key" "public_key" {
  name      = "${var.project_name}-ssh-key"
  namespace = var.namespace

  public_key = file(var.ssh_public_key_path)
}

resource "harvester_clusternetwork" "cluster-vlan" {
  count       = var.network_details.create ? 1 : 0
  name        = var.network_details.clusternetwork_name
  description = "Cluster VLAN managed by Dartboard's Harvester opentofu module"
}

resource "harvester_vlanconfig" "cluster-vlan-config" {
  count = var.network_details.create ? 1 : 0
  name  =  "${var.network_details.clusternetwork_name}-vlan-config"

  cluster_network_name = harvester_clusternetwork.cluster-vlan[0].name

  uplink {
    nics        = var.vlan_uplink.nics
    bond_mode   = var.vlan_uplink.bond_mode
    bond_miimon = var.vlan_uplink.bond_miimon
    mtu         = var.vlan_uplink.mtu
  }
}

resource "harvester_network" "this" {
  count = var.network_details.create ? 1 : 0
  depends_on  = [ harvester_vlanconfig.cluster-vlan-config ]
  name        = var.network_details.name
  namespace   = var.namespace
  description = "Harvester network managed by Dartboard's Harvester opentofu module"

  vlan_id = var.network_details.vlan_id
  cluster_network_name = harvester_clusternetwork.cluster-vlan[0].name
}

resource "harvester_image" "created" {
  count = var.create_image ? 1 : 0
  name = "${var.project_name}-opensuse156"
  namespace = var.namespace
  display_name = "${var.project_name}-opensuse156"
  source_type = "download"
  url = "https://download.opensuse.org/repositories/Cloud:/Images:/Leap_15.6/images/openSUSE-Leap-15.6.x86_64-NoCloud.qcow2"
}
