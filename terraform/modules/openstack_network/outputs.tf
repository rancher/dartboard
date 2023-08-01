output "private_network_id" {
  value = length(openstack_networking_network_v2.network) > 0 ? openstack_networking_network_v2.network[0].id : var.network_id
}

output "private_subnet_id" {
  value = openstack_networking_subnet_v2.main.id
}

output "bastion_public_name" {
  depends_on = [ module.bastion ]
  value = module.bastion.public_name
}
