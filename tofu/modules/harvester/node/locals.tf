locals {
  ssh_keys    = { for key in coalesce(try(var.node_module_variables.ssh_shared_public_keys, null), []) : "${key.namespace}/${key.name}" => var.network_config.ssh_keys_by_name["${key.namespace}/${key.name}"] }
  public_keys = compact(concat([var.network_config.ssh_public_key], [for key in local.ssh_keys : key.public_key]))
  ssh_key_ids = compact(concat([var.network_config.ssh_public_key_id], [for key in local.ssh_keys : key.id]))
  template_user_data = templatefile("${path.module}/user_data.yaml", {
    ssh_user = var.ssh_user
    password = var.node_module_variables.password
    ssh_keys = local.public_keys
  })
  wait_for_lease = var.network_config.wait_for_lease
  disks_map      = { for disk in var.node_module_variables.disks : disk.name => disk }

  private_network_interfaces = [for network in harvester_virtualmachine.this.network_interface[*] : {
    interface_name = network.interface_name
    ip_address     = network.ip_address
    } if !var.network_config.public && !strcontains(tostring(network.ip_address), ":")
  ]
  public_network_interfaces = [for network in harvester_virtualmachine.this.network_interface[*] : {
    interface_name = network.interface_name
    ip_address     = network.ip_address
    } if var.network_config.public && !strcontains(tostring(network.ip_address), ":")
  ]
  image_namespace = replace(lower(var.node_module_variables.image_namespace != null ? var.node_module_variables.image_namespace : var.network_config.namespace), "/[^a-z0-9-]/", "-") # Convert to valid Kubernetes name
}
