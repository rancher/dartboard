output "backend_variables" {
  value = {
    location: var.location,
    resource_group_name: azurerm_resource_group.rg.name,
    public_subnet_id: azurerm_subnet.public.id,
    private_subnet_id: azurerm_subnet.private.id,
    ssh_public_key_path: var.ssh_public_key_path,
    ssh_bastion_host: module.bastion.public_name,
    ssh_bastion_user: var.ssh_bastion_user,
    storage_account_uri: azurerm_storage_account.storage_account.primary_blob_endpoint,
  }
}
