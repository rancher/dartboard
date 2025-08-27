resource "aws_instance" "instance" {
  ami                    = var.node_module_variables.ami
  instance_type          = var.node_module_variables.instance_type
  availability_zone      = var.network_config.availability_zone
  key_name               = var.network_config.ssh_key_name
  subnet_id              = var.public ? var.network_config.public_subnet_id : var.network_config.private_subnet_id
  vpc_security_group_ids = distinct(concat([
    var.public ? var.network_config.public_security_group_id : var.network_config.private_security_group_id], [var.network_config.private_security_group_id], var.network_config.other_security_group_ids))

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
    host        = var.network_config.ssh_bastion_host == null ? aws_instance.instance.public_dns : aws_instance.instance.private_dns
    private_key = file(var.ssh_private_key_path)
    user        = var.ssh_user

    bastion_host        = var.network_config.ssh_bastion_host
    bastion_user        = var.network_config.ssh_bastion_user
    bastion_private_key = file(var.ssh_private_key_path)
    timeout             = "240s"
  }

  provisioner "file" {
    source      = "${path.module}/mount_ephemeral.sh"
    destination = "/tmp/mount_ephemeral.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/mount_ephemeral.sh",
      "sudo /tmp/mount_ephemeral.sh"
    ]
  }

  provisioner "remote-exec" {
    inline = var.host_configuration_commands
  }
}
