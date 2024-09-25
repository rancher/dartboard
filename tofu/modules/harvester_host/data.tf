data "harvester_image" "this" {
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
