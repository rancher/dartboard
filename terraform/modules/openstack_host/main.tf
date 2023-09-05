terraform {
  required_providers {
    openstack = {
      source = "terraform-provider-openstack/openstack"
    }
  }
}

resource "openstack_networking_port_v2" "port" {
  name                  = "${var.project_name}-${var.name}-port"
  network_id            = var.network_id
  admin_state_up        = "true"
  port_security_enabled = false
  security_group_ids    = null
  no_security_groups    = true

  fixed_ip {
    subnet_id = var.subnet_id
  }
}

// Setup a Floating IP if we need exposure on the Public network
resource "openstack_networking_floatingip_v2" "fip" {
  count = var.attach_floating_ip_from == null ? 0 : 1
  pool  = var.attach_floating_ip_from
}

resource "openstack_networking_floatingip_associate_v2" "fip_1" {
  count       = var.attach_floating_ip_from == null ? 0 : 1
  floating_ip = openstack_networking_floatingip_v2.fip[0].address
  port_id     = openstack_networking_port_v2.port.id
}

resource "openstack_compute_instance_v2" "instance" {
  name              = "${var.project_name}-${var.name}"
  image_id          = var.image
  flavor_name       = var.flavor
  key_pair          = var.keypair
  availability_zone = var.availability_zone
  user_data         = templatefile("${path.module}/user_data.yaml", {})

  network {
    port = openstack_networking_port_v2.port.id
  }
}

resource "null_resource" "host_configuration" {
  depends_on = [openstack_compute_instance_v2.instance]
  connection {
    host                = length(openstack_networking_floatingip_v2.fip) >= 1 ? openstack_networking_floatingip_v2.fip[0].address : openstack_compute_instance_v2.instance.access_ip_v4
    private_key         = file(var.ssh_private_key_path)
    user                = "root"
    bastion_host        = var.ssh_bastion_host
    bastion_user        = "root"
    bastion_private_key = file(var.ssh_private_key_path)
    timeout             = "120s"
  }

  provisioner "remote-exec" {
    inline = var.host_configuration_commands
  }
}
