
output "id" {
  value = "id:${var.name}"
}

output "private_name" {
  value = var.fqdn
}

output "public_name" {
  value = var.fqdn
}

resource "local_file" "ssh_script" {
  content = <<-EOT
    #!/bin/sh
    ssh -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" \
      ${var.ssh_user}@${var.fqdn} \
      $@
  EOT

  filename = "${path.module}/../../../config/ssh-to-${var.name}.sh"
}

output "name" {
  value = var.name
}

output "ssh_script_filename" {
  value = abspath(local_file.ssh_script.filename)
}
