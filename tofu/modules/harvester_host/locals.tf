locals {
  nonexistent_cloudinit_secrets     = [for cloudinit in var.cloudinit_secrets : cloudinit if length(cloudinit.user_data) > 0]
  nonexistent_cloudinit_secrets_map = { for cloudinit in local.nonexistent_cloudinit_secrets : cloudinit.name => cloudinit }
  existing_cloudinit_secrets        = [for cloudinit in var.cloudinit_secrets : cloudinit if !contains(local.nonexistent_cloudinit_secrets, cloudinit)]
  existing_cloudinit_secrets_map    = { for cloudinit in local.existing_cloudinit_secrets : cloudinit.name => cloudinit }
  default_init                      = <<EOT
#cloud-config
users:
  - default
  - ${var.user}
password: ${var.password}
disable_root: false
chpasswd:
  expire: false
  users:
    - {name: ${var.user}, password: ${var.password}, type: text}
EOT
  guest_agent_init                  = <<EOT
package_update: true
packages:
  - qemu-guest-agent
  - iptables
runcmd:
  - - systemctl
    - enable
    - --now
    - qemu-guest-agent.service
EOT
  ssh_authorized_keys               = <<EOT
ssh_authorized_keys:
%%{ for public_key in ssh_keys ~}
  - >-
    $${public_key}
%%{ endfor ~}
EOT
  public_keys                       = compact([var.ssh_public_key, try(data.harvester_ssh_key.shared[0].public_key, null)])
  authorized_keys_userdata          = templatestring(local.ssh_authorized_keys, { ssh_keys = local.public_keys })
  all_user_data = join("\n", compact(flatten(concat(
    [for secret in harvester_cloudinit_secret.this[*] : secret.user_data if length(secret) > 0],
    [for secret in data.harvester_cloudinit_secret.this[*] : secret.user_data if length(secret) > 0]
  ))))
  wait_for_lease       = contains(var.networks[*].wait_for_lease, true)
  cloud_init_user_data = local.wait_for_lease ? format("%s%s\n%s%s\n", local.default_init, local.all_user_data, local.guest_agent_init, local.authorized_keys_userdata) : format("%s%s", local.default_init, local.all_user_data)
  networks_map         = { for network in var.networks : network.name => network }
  disks_map            = { for disk in var.disks : disk.name => disk }

  private_network_interfaces = [for network in harvester_virtualmachine.this.network_interface[*] : {
    interface_name = network.interface_name
    ip_address     = network.ip_address
    } if !local.networks_map[network.name].public
  ]
  public_network_interfaces = [for network in harvester_virtualmachine.this.network_interface[*] : {
    interface_name = network.interface_name
    ip_address     = network.ip_address
    } if local.networks_map[network.name].public
  ]
}
