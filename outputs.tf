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
