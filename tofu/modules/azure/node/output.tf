output "name" {
  value = var.name
}

output "private_name" {
  value = "${azurerm_linux_virtual_machine.main.private_ip_address}.sslip.io"
}

output "private_ip" {
  value = azurerm_linux_virtual_machine.main.private_ip_address
}

output "public_name" {
  value = azurerm_linux_virtual_machine.main.public_ip_address
}
