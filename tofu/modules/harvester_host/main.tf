resource "harvester_virtualmachine" "this" {
  name      = "${var.project_name}-${var.name}"
  namespace = var.namespace
  hostname  = var.name

  tags = merge({
    ssh-user = var.user
    Project  = var.project_name
  }, var.tags)

  cpu    = var.cpu
  memory = "${var.memory}Gi"

  efi         = var.efi
  secure_boot = var.efi ? var.secure_boot : false

  dynamic "network_interface" {
    for_each = data.harvester_network.this
    content {
      name           = network_interface.value.name
      network_name   = network_interface.value.id
      type           = local.networks_map[network_interface.value.name].interface_type
      model          = local.networks_map[network_interface.value.name].interface_model
      wait_for_lease = local.networks_map[network_interface.value.name].wait_for_lease
    }
  }

  dynamic "disk" {
    for_each = local.disks_map
    content {
      name       = disk.value.name
      type       = disk.value.type
      size       = "${disk.value.size}Gi"
      bus        = disk.value.bus
      image      = index(var.disks, disk.value) == 0 ? (
              var.image_name == null ? var.default_image_id : data.harvester_image.this[0].id
      ) : null
      boot_order = index(var.disks, disk.value) + 1 //boot_order starts at 1, while the index() function is 0-based
      auto_delete = true
    }
  }

  ssh_keys = compact([var.ssh_public_key_id, try(data.harvester_ssh_key.shared[0].id, null)])

  # Default "USB Tablet" config for VNC usage
  input {
    name = "tablet"
    type = "tablet"
    bus  = "usb"
  }

  cloudinit {
    user_data = local.cloud_init_user_data
  }

  // Allow for more than the default time for VM destruction
  // Uncomment once https://github.com/harvester/harvester/issues/6647 is resolved
  # timeouts {
  #   delete = "15m"
  # }
}

resource "null_resource" "cloud_init_wait" {
  depends_on = [harvester_virtualmachine.this]
  # # IMPORTANT: We need to wait for cloud-init on Harvester VMs to complete
  provisioner "remote-exec" {
    connection {
      host        = local.public_network_interfaces[0].ip_address
      private_key = var.ssh_private_key_path != null ? file(var.ssh_private_key_path) : null
      user        = var.user

      bastion_host        = var.ssh_bastion_host
      bastion_user        = var.ssh_bastion_user
      bastion_private_key = var.ssh_bastion_key_path != null ? file(var.ssh_bastion_key_path) : null
      bastion_port        = 22
      timeout             = "3m"
    }
    inline = [
      "echo 'Waiting for cloud-init to complete...'",
      "cloud-init status --wait > /dev/null",
      "echo 'Completed cloud-init!'",
    ]
  }
}

resource "harvester_cloudinit_secret" "this" {
  for_each  = local.nonexistent_cloudinit_secrets_map != null ? local.nonexistent_cloudinit_secrets_map : {}
  name      = each.value.name
  namespace = each.value.namespace
  user_data = each.value.user_data
}

resource "null_resource" "host_configuration" {
  depends_on = [null_resource.cloud_init_wait]

  connection {
    host        = local.public_network_interfaces[0].ip_address
    private_key = var.ssh_private_key_path != null ? file(var.ssh_private_key_path) : null
    user        = var.user

    bastion_host        = var.ssh_bastion_host
    bastion_user        = var.ssh_bastion_user
    bastion_private_key = var.ssh_bastion_key_path != null ? file(var.ssh_bastion_key_path) : null
    bastion_port        = 22
    timeout             = "120s"
  }
  provisioner "remote-exec" {
    inline = var.host_configuration_commands
  }
}

module "ssh_access" {
  depends_on = [null_resource.host_configuration]

  source = "../ssh_access"
  name   = var.name

  ssh_bastion_host     = var.ssh_bastion_host
  ssh_tunnels          = var.ssh_tunnels
  private_name         = local.public_network_interfaces[0].ip_address
  public_name          = local.public_network_interfaces[0].ip_address
  ssh_user             = var.user
  ssh_bastion_user     = var.ssh_bastion_user
  ssh_private_key_path = var.ssh_private_key_path
}
