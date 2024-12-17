output "private_name" {
  value = "${azurerm_linux_virtual_machine.main.private_ip_address}.sslip.io"
}

output "private_ip" {
  value = azurerm_linux_virtual_machine.main.private_ip_address
}

output "public_name" {
  value = azurerm_linux_virtual_machine.main.public_ip_address
}

output "name" {
  value = var.name
}

output "ssh_script_filename" {
  value = abspath(module.ssh_access.ssh_script_filename)
}
