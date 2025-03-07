data "harvester_image" "this" {
  count        = var.node_module_variables.image_name != null && var.node_module_variables.image_namespace != null ? 1 : 0
  display_name = var.node_module_variables.image_name
  namespace    = local.image_namespace
}

data "harvester_ssh_key" "shared" {
  for_each = var.node_module_variables.ssh_shared_public_keys != null ? {
    for i, key in var.node_module_variables.ssh_shared_public_keys :
    key.name => key
  } : {}
  name      = each.value.name
  namespace = each.value.namespace
}
