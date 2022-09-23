resource "local_file" "rancher_kubeconfig" {
  content = yamlencode({
    apiVersion = "v1"
    clusters = [{
      cluster = {
        certificate-authority-data = base64encode(module.secrets.cluster_ca_certificate)
        server                     = "https://${module.bastion.public_names[0]}:6443"
      }
      name = module.bastion.public_name
    }]
    contexts = [{
      context = {
        cluster = module.bastion.public_name
        user : "master-user"
      }
      name = module.bastion.public_name
    }]
    current-context = module.bastion.public_name
    kind            = "Config"
    preferences     = {}
    users = [{
      user = {
        client-certificate-data : base64encode(module.secrets.master_user_cert)
        client-key-data : base64encode(module.secrets.master_user_key)
      }
      name : "master-user"
    }]
  })

  filename = "${path.module}/config/rancher.yaml"
}

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
  logins = [for i in range(length(module.nodes.private_names)) : "./config/login_node-${i+1}.sh"]
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
