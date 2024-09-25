output "ssh_public_key_id" {
  value = harvester_ssh_key.public_key.id
}

output "ssh_public_key" {
  value = harvester_ssh_key.public_key.public_key
}
