terraform {
  required_providers {
    harvester = {
      source = "harvester/harvester"
    }
  }
}

resource "harvester_virtualmachine" "this" {
  name      = "${var.project_name}-${var.name}"
  namespace = var.network_config.namespace
  hostname  = var.name

  tags = merge({
    ssh-user = var.ssh_user
    Project  = var.project_name
  }, var.node_module_variables.tags)

  cpu    = var.node_module_variables.cpu
  memory = "${var.node_module_variables.memory}Gi"

  efi         = coalesce(var.node_module_variables.efi, false)
  secure_boot = coalesce(var.node_module_variables.efi, false) ? coalesce(var.node_module_variables.secure_boot, false) : false

  network_interface {
    name = var.network_config.name
    network_name = var.network_config.id
    type           = var.network_config.interface_type
    model          = var.network_config.interface_model
    wait_for_lease = var.network_config.wait_for_lease
  }

  dynamic "disk" {
    for_each = local.disks_map
    content {
      name       = disk.value.name
      type       = disk.value.type
      size       = "${disk.value.size}Gi"
      bus        = disk.value.bus
      image      = index(var.node_module_variables.disks, disk.value) == 0 ? (
        var.node_module_variables.image_name != null && var.node_module_variables.image_namespace != null ? data.harvester_image.this[0].id : var.image_id
      ) : null
      boot_order = index(var.node_module_variables.disks, disk.value) + 1 //boot_order starts at 1, while the index() function is 0-based
      auto_delete = true
    }
  }

  ssh_keys = compact([var.network_config.ssh_public_key_id, try(data.harvester_ssh_key.shared[0].id, null)])

  # Default "USB Tablet" config for VNC usage
  input {
    name = "tablet"
    type = "tablet"
    bus  = "usb"
  }

  cloudinit {
    user_data = local.template_user_data
  }

  // Allow for more than the default time for VM destruction
  timeouts {
    delete = "15m"
    create = "5m"
  }
}

resource "null_resource" "host_configuration" {
  connection {
    host        = local.public_network_interfaces[0].ip_address
    private_key = var.ssh_private_key_path != null ? file(var.ssh_private_key_path) : null
    user        = var.ssh_user

    bastion_host        = var.network_config.ssh_bastion_host
    bastion_user        = var.network_config.ssh_bastion_user
    bastion_private_key = var.network_config.ssh_bastion_key_path != null ? file(var.network_config.ssh_bastion_key_path) : null
    bastion_port        = 22
    timeout             = "5m"
  }
  provisioner "remote-exec" {
    inline = var.host_configuration_commands
  }
}
