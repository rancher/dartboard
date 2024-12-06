resource "aws_instance" "instance" {
  ami                    = var.backend_variables.ami
  instance_type          = var.backend_variables.instance_type
  availability_zone      = var.backend_network_variables.availability_zone
  key_name               = var.backend_network_variables.ssh_key_name
  subnet_id              = var.public ? var.backend_network_variables.public_subnet_id : var.backend_network_variables.private_subnet_id
  vpc_security_group_ids = [var.public ? var.backend_network_variables.public_security_group_id : var.backend_network_variables.private_security_group_id]

  root_block_device {
    volume_size = var.backend_variables.root_volume_size_gb
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
    host        = var.backend_network_variables.ssh_bastion_host == null ? aws_instance.instance.public_dns : aws_instance.instance.private_dns
    private_key = file(var.ssh_private_key_path)
    user        = var.ssh_user

    bastion_host        = var.backend_network_variables.ssh_bastion_host
    bastion_user        = var.backend_network_variables.ssh_bastion_user
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

module "ssh_access" {
  depends_on = [null_resource.host_configuration]

  source = "../../ssh/access"
  name   = var.name

  ssh_bastion_host     = var.backend_network_variables.ssh_bastion_host
  ssh_tunnels          = var.ssh_tunnels
  private_name         = aws_instance.instance.private_dns
  public_name          = aws_instance.instance.public_dns
  ssh_user             = var.ssh_user
  ssh_bastion_user     = var.backend_network_variables.ssh_bastion_user
  ssh_private_key_path = var.ssh_private_key_path
}
