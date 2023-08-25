
output "id" {
  value = "id:${var.name}"
}

output "private_name" {
  value = "pri.${replace(var.ssh_addr, ".", "-")}.sslip.io"
}

output "private_ip" {
  value = var.ssh_addr
}

output "public_name" {
  depends_on = [null_resource.host_configuration]
  value      = "pub.${replace(var.ssh_addr, ".", "-")}.sslip.io"
}

resource "local_file" "ssh_script" {
  content = <<-EOT
    #!/bin/sh
    ssh -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" \
      ${var.ssh_user}@${var.ssh_addr} \
      $@
  EOT

  filename = "${path.module}/../../../config/ssh-to-${var.name}.sh"
}


output "name" {
  value = var.name
}

output "addr" {
  value = var.ssh_addr
}

output "ssh_script_filename" {
  value = abspath(local_file.ssh_script.filename)
}
