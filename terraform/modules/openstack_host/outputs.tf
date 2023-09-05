output "id" {
  value = openstack_compute_instance_v2.instance.id
}

output "private_name" {
  value = openstack_compute_instance_v2.instance.access_ip_v4
}

output "private_ip" {
  value = openstack_compute_instance_v2.instance.access_ip_v4
}

output "public_name" {
  depends_on = [null_resource.host_configuration]
  value      = length(openstack_networking_floatingip_v2.fip) >= 1 ? "${openstack_networking_floatingip_v2.fip[0].address}.${var.ip_wildcard_resolver_domain}" : openstack_compute_instance_v2.instance.access_ip_v4
}

resource "local_file" "ssh_script" {
  content  = <<-EOT
    #!/bin/sh
    ssh -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" \
      %{if var.ssh_bastion_host != null~}
      -o ProxyCommand="ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -W %h:%p root@${var.ssh_bastion_host}" root@${openstack_compute_instance_v2.instance.access_ip_v4} \
      %{else~}
      root@${openstack_networking_floatingip_v2.fip[0].address} \
      %{endif~}
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
