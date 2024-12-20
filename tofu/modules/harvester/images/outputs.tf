output "opensuse156_id" {
  value = var.create ? harvester_image.opensuse156[0].id : null
}
