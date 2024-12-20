data "harvester_image" "this" {
  count = var.node_module_variables.image_name != null && var.node_module_variables.image_namespace != null ? 1 : 0
  display_name = var.node_module_variables.image_name
  namespace    = var.node_module_variables.image_namespace
}

data "harvester_network" "this" {
  for_each  = local.networks_map
  name      = each.value.name
  namespace = each.value.namespace
}

data "harvester_cloudinit_secret" "this" {
  for_each  = local.existing_cloudinit_secrets_map != null ? local.existing_cloudinit_secrets_map : {}
  name      = each.value.name
  namespace = each.value.namespace
}

data "harvester_ssh_key" "shared" {
  for_each = {
    for i, key in var.node_module_variables.ssh_shared_public_keys:
    key.name => key
  }
  name = each.value.name
  namespace = each.value.namespace
}
