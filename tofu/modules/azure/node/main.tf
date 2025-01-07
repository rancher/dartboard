resource "azurerm_public_ip" "public" {
  count               = var.public ? 1 : 0
  name                = "${var.project_name}-public-ip"
  location            = var.network_config.location
  resource_group_name = var.network_config.resource_group_name
  allocation_method   = "Static"
}

resource "azurerm_network_interface" "nic" {
  name                = "${var.project_name}-${var.name}"
  resource_group_name = var.network_config.resource_group_name
  location            = var.network_config.location

  ip_configuration {
    name                          = "internal"
    subnet_id                     = var.public ? var.network_config.public_subnet_id : var.network_config.private_subnet_id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = var.public ? azurerm_public_ip.public[0].id : null
  }
}

resource "azurerm_linux_virtual_machine" "main" {
  name                  = "${var.project_name}-${var.name}"
  resource_group_name   = var.network_config.resource_group_name
  location              = var.network_config.location
  size                  = var.node_module_variables.size
  priority              = var.node_module_variables.is_spot ? "Spot" : "Regular"
  eviction_policy       = var.node_module_variables.is_spot ? "Deallocate" : null
  admin_username        = var.ssh_user
  network_interface_ids = [azurerm_network_interface.nic.id]

  admin_ssh_key {
    username   = var.ssh_user
    public_key = file(var.network_config.ssh_public_key_path)
  }

  source_image_reference {
    publisher = var.node_module_variables.os_image.publisher
    offer     = var.node_module_variables.os_image.offer
    sku       = var.node_module_variables.os_image.sku
    version   = var.node_module_variables.os_image.version
  }

  os_disk {
    storage_account_type = var.node_module_variables.os_ephemeral_disk ? "Standard_LRS" : var.node_module_variables.os_disk_type
    disk_size_gb         = var.node_module_variables.os_ephemeral_disk ? null : var.node_module_variables.os_disk_size
    caching              = var.node_module_variables.os_ephemeral_disk ? "ReadOnly" : "ReadWrite"

    dynamic "diff_disk_settings" {
      for_each = var.node_module_variables.os_ephemeral_disk ? [1] : []
      content {
        option    = "Local"
        placement = "ResourceDisk"
      }
    }
  }

  dynamic "boot_diagnostics" {
    for_each = var.network_config.storage_account_uri != null ? [1] : []
    content {
      storage_account_uri = var.network_config.storage_account_uri
    }
  }
}

resource "null_resource" "host_configuration" {
  depends_on = [azurerm_linux_virtual_machine.main]

  connection {
    host = coalesce(azurerm_linux_virtual_machine.main.public_ip_address,
    azurerm_linux_virtual_machine.main.private_ip_address)
    private_key = file(var.ssh_private_key_path)
    user        = var.ssh_user

    bastion_host        = var.network_config.ssh_bastion_host
    bastion_user        = var.network_config.ssh_bastion_user
    bastion_private_key = file(var.ssh_private_key_path)
    timeout             = "120s"
  }

  provisioner "remote-exec" {
    script = "${path.module}/mount_ephemeral.sh"
  }

  provisioner "remote-exec" {
    inline = var.host_configuration_commands
  }
}
