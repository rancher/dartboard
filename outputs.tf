resource "local_file" "login_bastion" {
  content = <<-EOT
    #!/bin/sh
    ssh -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" root@${module.bastion.public_name} $@
  EOT

  filename = "${path.module}/config/login_bastion.sh"
}

resource "local_file" "login_node" {
  for_each = {
    for i, name in module.nodes.private_names : i => name
  }

  filename = "${path.module}/config/login_node-${each.key + 1}.sh"

  content = <<-EOT
    #!/bin/sh
    ssh -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" -J root@${module.bastion.public_name} root@${each.value} $@
  EOT
}

locals {
  logins = [for i in range(length(module.nodes.private_names)) : "./config/login_node-${i + 1}.sh"]
}

output "rancher_help" {
  value = <<-EOT

    To reach the Rancher UI use:

      https://${module.bastion.public_name}

    To reach the bastion host use:

      ./config/login_bastion.sh

    To reach the Rancher cluster API use:

      kubectl --kubeconfig ./config/rancher.yaml
      helm --kubeconfig ./config/rancher.yaml
      k9s --kubeconfig ./config/rancher.yaml

    To reach downstream cluster nodes use:

      ${join("\n  ", local.logins)}
  EOT
}
