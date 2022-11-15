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

  user_data = templatefile("${path.module}/user_data.yaml", {})

  # WORKAROUND: ephemeral block devices are defined in any case
  # they will only be used for instance types that provide them
  ephemeral_block_device {
    device_name  = "xvdb"
    virtual_name = "ephemeral0"
  }

  ephemeral_block_device {
    device_name  = "xvdc"
    virtual_name = "ephemeral1"
  }

  tags = {
    Project = var.project_name
    Name    = "${var.project_name}-${var.name}"
  }
}

resource "null_resource" "host_configuration" {
  depends_on = [aws_instance.instance]

  connection {
    host        = coalesce(aws_instance.instance.public_dns, aws_instance.instance.private_dns)
    private_key = file(var.ssh_private_key_path)
    user        = "root"

    bastion_host        = var.ssh_bastion_host
    bastion_user        = "root"
    bastion_private_key = file(var.ssh_private_key_path)
    timeout             = "120s"
  }

  provisioner "remote-exec" {
    inline = [
      "cat /etc/os-release",
    ]
  }
}

resource "local_file" "open_tunnels" {
  count = length(var.ssh_tunnels) > 0 ? 1 : 0
  content = templatefile("${path.module}/open-tunnels-to.sh", {
    ssh_bastion_host = var.ssh_bastion_host,
    ssh_tunnels      = var.ssh_tunnels,
    private_name     = aws_instance.instance.private_dns
  })

  filename = "${path.module}/../../config/open-tunnels-to-${var.name}.sh"
}

resource "null_resource" "open_tunnels" {
  count      = length(var.ssh_tunnels) > 0 ? 1 : 0
  depends_on = [null_resource.host_configuration]
  provisioner "local-exec" {
    interpreter = ["bash", "-c"]
    command     = local_file.open_tunnels[0].filename
  }
  triggers = {
    always_run = timestamp()
  }
}

output "id" {
  value = aws_instance.instance.id
}

output "private_name" {
  value = aws_instance.instance.private_dns
}

output "public_name" {
  value = aws_instance.instance.public_dns
}
