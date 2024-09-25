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

data "harvester_ssh_key" "shared" {
  count = var.ssh_shared_public_key != null ? 1 : 0
  name = var.ssh_shared_public_key.name
  namespace = var.ssh_shared_public_key.namespace
}
