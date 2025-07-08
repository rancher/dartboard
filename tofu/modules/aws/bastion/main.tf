resource "aws_instance" "instance" {
  ami                    = var.node_module_variables.ami
  instance_type          = var.node_module_variables.instance_type
  availability_zone      = var.network_config.availability_zone
  key_name               = var.network_config.ssh_key_name
  subnet_id              = var.public ? var.network_config.public_subnet_id : var.network_config.private_subnet_id
  vpc_security_group_ids = distinct(concat([
    var.public ? var.network_config.public_security_group_id : var.network_config.private_security_group_id], var.network_config.other_security_group_ids))

  root_block_device {
    volume_size = var.node_module_variables.root_volume_size_gb
  }

  user_data = templatefile("${path.module}/user_data.yaml", { ssh_user = var.ssh_user })

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-${var.name}"
  }
}

resource "null_resource" "host_configuration" {
  depends_on = [aws_instance.instance]

  connection {
    host        = aws_instance.instance.public_dns
    private_key = file(var.ssh_private_key_path)
    user        = var.ssh_user

    bastion_private_key = file(var.ssh_private_key_path)
    timeout             = "240s"
  }

  provisioner "file" {
    source      = "${path.module}/mount_ephemeral.sh"
    destination = "/tmp/mount_ephemeral.sh"
  }

  provisioner "file" {
    source      = "${path.module}/dbus_max.connections.conf"
    destination = "/etc/dbus-1/system.d/max.connections.conf"
  }

  provisioner "file" {
    source      = "${path.module}/sshd_max-startups.conf"
    destination = "/etc/ssh/sshd_config.d/90-max-startups.conf"
  }

  provisioner "file" {
    source      = "${path.module}/user-slice.conf"
    destination = "/usr/lib/systemd/system/user-.slice.d/10-defaults.conf"
  }


  provisioner "file" {
    source      = "${path.module}/logind.conf"
    destination = "/etc/systemd/logind.conf.d/override.conf"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/mount_ephemeral.sh",
      "sudo /tmp/mount_ephemeral.sh",
      "systemctl reload sshd",
    ]
  }

  provisioner "remote-exec" {
    inline = var.host_configuration_commands
  }
}

resource "local_file" "ssh_script" {
  content = <<-EOT
    #!/bin/sh
    ssh -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" \
      -i ${var.ssh_private_key_path} \
      ${var.ssh_user}@${aws_instance.instance.public_dns} \
      $@
  EOT

  filename = "${path.root}/${terraform.workspace}_config/ssh-to-${var.name}.sh"
}

