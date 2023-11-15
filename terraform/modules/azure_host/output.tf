output "private_name" {
  value = azurerm_linux_virtual_machine.main.private_ip_address
}

output "public_name" {
  value = azurerm_linux_virtual_machine.main.public_ip_address
}

output "name" {
  value = var.name
}

resource "local_file" "ssh_script" {
  content = <<-EOT
    #!/bin/sh
    ssh -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" \
      %{if var.ssh_bastion_host != null~}
      -o ProxyCommand="ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -W %h:%p ${var.admin_username}@${var.ssh_bastion_host}" root@${azurerm_linux_virtual_machine.main.private_ip_address} \
      %{else~}
      ${var.admin_username}@${azurerm_linux_virtual_machine.main.public_ip_address} \
      %{endif~}
      $@
  EOT

  filename = "${path.module}/../../../config/ssh-to-${var.name}.sh"
}
output "ssh_script_filename" {
  value = abspath(local_file.ssh_script.filename)
}