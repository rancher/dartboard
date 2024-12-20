output "ssh_public_key_id" {
  value = harvester_ssh_key.public_key.id
}

output "ssh_public_key" {
  value = harvester_ssh_key.public_key.public_key
}

output "name" {
  value = var.create ? harvester_network.this[0].name : data.harvester_network.this[0].name
}

output "namespace" {
  value = var.create ? harvester_network.this[0].namespace : data.harvester_network.this[0].namespace
}

output "id" {
    value = var.create ? harvester_network.this[0].id : data.harvester_network.this[0].id
}
