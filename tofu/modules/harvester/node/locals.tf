locals {
  public_keys                       = compact([var.node_module_variables.ssh_public_key, try(data.harvester_ssh_key.shared[0].public_key, null)])
  # authorized_keys_userdata          = templatestring(local.ssh_authorized_keys, { ssh_keys = local.public_keys })
  template_user_data = templatefile("${path.module}/user_data.yaml", {
      ssh_user = var.ssh_user
      password = var.node_module_variables.password
      ssh_keys = local.public_keys
     })
  cloudinit_user_data = join("\n", compact(flatten(concat(
    [for secret in harvester_cloudinit_secret.this[*] : secret.user_data if length(secret) > 0],
    [for secret in data.harvester_cloudinit_secret.this[*] : secret.user_data if length(secret) > 0]
  ))))
  all_user_data = format("%s%s", local.template_user_data, local.cloudinit_user_data)
  wait_for_lease       = var.network_config.wait_for_lease
  disks_map            = { for disk in var.node_module_variables.disks : disk.name => disk }

  private_network_interfaces = [for network in harvester_virtualmachine.this.network_interface[*] : {
    interface_name = network.interface_name
    ip_address     = network.ip_address
    } if !var.node_module_variables.public
  ]
  public_network_interfaces = [for network in harvester_virtualmachine.this.network_interface[*] : {
    interface_name = network.interface_name
    ip_address     = network.ip_address
    } if var.node_module_variables.public
  ]
}
