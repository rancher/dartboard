resource "aws_instance" "instance" {
  ami                    = var.ami
  instance_type          = var.instance_type
  availability_zone      = var.availability_zone
  key_name               = var.ssh_key_name
  subnet_id              = var.subnet_id
  vpc_security_group_ids = [var.vpc_security_group_id]

  root_block_device {
    volume_size = var.root_volume_size_gb
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
    host        = var.ssh_bastion_host == null ? aws_instance.instance.public_dns : aws_instance.instance.private_dns
    private_key = file(var.ssh_private_key_path)
    user        = var.ssh_user

    bastion_host        = var.ssh_bastion_host
    bastion_user        = var.ssh_bastion_user
    bastion_private_key = file(var.ssh_private_key_path)
    timeout             = "120s"
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

  source = "../ssh_access"
  name   = var.name

  ssh_bastion_host     = var.ssh_bastion_host
  ssh_tunnels          = var.ssh_tunnels
  private_name         = aws_instance.instance.private_dns
  public_name          = aws_instance.instance.public_dns
  ssh_user             = var.ssh_user
  ssh_bastion_user     = var.ssh_bastion_user
  ssh_private_key_path = var.ssh_private_key_path
}
