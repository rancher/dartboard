resource "null_resource" "host_configuration" {
  connection {
    host        = var.fqdn
    user        = var.ssh_user
    private_key = file(var.ssh_private_key_path)

    timeout = "120s"
  }

  provisioner "remote-exec" {
    inline = var.host_configuration_commands
  }

  provisioner "remote-exec" {
    inline = var.host_configuration_commands
  }
}

