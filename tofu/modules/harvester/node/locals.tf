locals {
  public_keys                       = compact([var.network_config.ssh_public_key, try(data.harvester_ssh_key.shared[0].public_key, null)])
  template_user_data = templatefile("${path.module}/user_data.yaml", {
      ssh_user = var.ssh_user
      password = var.node_module_variables.password
      ssh_keys = local.public_keys
     })
  wait_for_lease       = var.network_config.wait_for_lease
  disks_map            = { for disk in var.node_module_variables.disks : disk.name => disk }

  private_network_interfaces = [for network in harvester_virtualmachine.this.network_interface[*] : {
    interface_name = network.interface_name
    ip_address     = network.ip_address
    } if !var.network_config.public
  ]
  public_network_interfaces = [for network in harvester_virtualmachine.this.network_interface[*] : {
    interface_name = network.interface_name
    ip_address     = network.ip_address
    } if var.network_config.public
  ]
  image_namespace = replace(lower(var.node_module_variables.image_namespace != null ? var.node_module_variables.image_namespace : var.network_config.namespace), "/[^a-z0-9-]/", "-")  # Convert to valid Kubernetes name
}
