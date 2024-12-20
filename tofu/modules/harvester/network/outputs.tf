output "config" {
  value = {
    namespace : var.namespace,
    ssh_public_key_id : harvester_ssh_key.public_key.id,
    ssh_public_key : harvester_ssh_key.public_key.public_key,


    name : var.network_details.name
    clusternetwork_name : var.network_details.clusternetwork_name
    namespace : var.network_details.create ? var.network_details.namespace : data.harvester_network.this[0].namespace
    interface_type : var.network_details.interace_type
    interface_model : var.network_details.interface_model
    public : var.network_details.public
    wait_for_lease : var.network_details.wait_for_lease

    opensuse156_id = var.create_image ? harvester_image.opensuse156[0].id : null
    ssh_bastion_host : var.ssh_bastion_host
    ssh_bastion_user : var.ssh_bastion_user
    ssh_bastion_key_path : var.ssh_bastion_key_path
  }
}
