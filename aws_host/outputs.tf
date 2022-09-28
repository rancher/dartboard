resource "local_file" "login_script" {
  content = <<-EOT
    #!/bin/sh
    ssh -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" \
      %{if var.ssh_bastion_host != null}-o ProxyCommand="ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -W %h:%p root@${var.ssh_bastion_host}"%{endif}\
      root@${aws_instance.instance.private_dns} $@
  EOT

  filename = "${path.module}/../config/ssh-to-${var.name}.sh"
}
