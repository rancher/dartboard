data "harvester_image" "this" {
  count = var.image_name != null && var.image_namespace != null ? 1 : 0
  display_name = var.image_name
  namespace    = var.image_namespace
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
    for i, key in var.ssh_shared_public_keys:
    key.name => key
  }
  name = each.value.name
  namespace = each.value.namespace
}
