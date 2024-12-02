resource "azurerm_network_interface" "nic" {
  name                = "${var.project_name}-${var.name}"
  resource_group_name = var.resource_group_name
  location            = var.location

  ip_configuration {
    name                          = "internal"
    subnet_id                     = var.subnet_id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = var.public_ip_address_id
  }
}

resource "azurerm_linux_virtual_machine" "main" {
  name                  = "${var.project_name}-${var.name}"
  resource_group_name   = var.resource_group_name
  location              = var.location
  size                  = var.size
  priority              = var.is_spot ? "Spot" : "Regular"
  eviction_policy       = var.is_spot ? "Deallocate" : null
  admin_username        = var.ssh_user
  network_interface_ids = [azurerm_network_interface.nic.id]

  admin_ssh_key {
    username   = var.ssh_user
    public_key = file(var.ssh_public_key_path)
  }

  source_image_reference {
    publisher = var.os_image.publisher
    offer     = var.os_image.offer
    sku       = var.os_image.sku
    version   = var.os_image.version
  }

  os_disk {
    storage_account_type = var.os_ephemeral_disk ? "Standard_LRS" : var.os_disk_type
    disk_size_gb         = var.os_ephemeral_disk ? null : var.os_disk_size
    caching              = var.os_ephemeral_disk ? "ReadOnly" : "ReadWrite"

    dynamic "diff_disk_settings" {
      for_each = var.os_ephemeral_disk ? [1] : []
      content {
        option    = "Local"
        placement = "ResourceDisk"
      }
    }
  }

  dynamic "boot_diagnostics" {
    for_each = var.storage_account_uri != null ? [1] : []
    content {
      storage_account_uri = var.storage_account_uri
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

    bastion_host        = var.ssh_bastion_host
    bastion_user        = var.ssh_user
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

module "ssh_access" {
  depends_on = [null_resource.host_configuration]

  source = "../../ssh_access"
  name   = var.name

  ssh_bastion_host     = var.ssh_bastion_host
  ssh_tunnels          = var.ssh_tunnels
  private_name         = azurerm_linux_virtual_machine.main.private_ip_address
  public_name          = azurerm_linux_virtual_machine.main.public_ip_address
  ssh_user             = var.ssh_user
  ssh_private_key_path = var.ssh_private_key_path
}
