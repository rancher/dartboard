output "public_subnet_id" {
  value = azurerm_subnet.public.id
}

output "private_subnet_id" {
  value = azurerm_subnet.private.id
}

output "bastion_public_name" {
  value = azurerm_public_ip.public.ip_address
}