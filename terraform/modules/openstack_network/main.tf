terraform {
  required_providers {
    openstack = {
      source = "terraform-provider-openstack/openstack"
    }
  }
}

resource "openstack_networking_network_v2" "network" {
  count          = var.network_id == null ? 1 : 0
  name           = "${var.project_name}-network"
  admin_state_up = "true"
}

resource "openstack_networking_router_v2" "gateway" {
  name                    = "${var.project_name}-router"
  admin_state_up          = "true"
  external_network_id     = var.external_network_id
  availability_zone_hints = [var.availability_zone]
}

resource "openstack_networking_subnet_v2" "main" {
  name            = "${var.project_name}-internal-network"
  network_id      = length(openstack_networking_network_v2.network) > 0 ? openstack_networking_network_v2.network[0].id : var.network_id
  cidr            = var.subnet_cidr
  ip_version      = 4
  dns_nameservers = var.dns_nameservers
}

resource "openstack_networking_router_interface_v2" "gateway_subnet" {
  router_id = openstack_networking_router_v2.gateway.id
  subnet_id = openstack_networking_subnet_v2.main.id
}

module "bastion" {
  depends_on              = [openstack_networking_router_interface_v2.gateway_subnet]
  source                  = "../openstack_host"
  availability_zone       = var.availability_zone
  project_name            = var.project_name
  flavor                  = var.bastion_flavor
  image                   = var.bastion_image
  ssh_private_key_path    = var.ssh_private_key_path
  name                    = "bastion"
  keypair                 = var.keypair
  network_id              = length(openstack_networking_network_v2.network) > 0 ? openstack_networking_network_v2.network[0].id : var.network_id
  subnet_id               = openstack_networking_subnet_v2.main.id
  attach_floating_ip_from = var.floating_ip_pool_ext
}
