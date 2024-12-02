locals {
  local_kubernetes_api_url = "https://${var.sans[0]}:${var.local_kubernetes_api_port}"
}

resource "local_file" "kubeconfig" {
  content = yamlencode({
    apiVersion = "v1"
    clusters = [
      {
        cluster = {
          certificate-authority-data = base64encode(tls_self_signed_cert.server_ca_cert.cert_pem)
          server                     = local.local_kubernetes_api_url
        }
        name = var.name
      }
    ]
    contexts = [
      {
        context = {
          cluster = var.name
          user : "master-user"
        }
        name = var.name
      }
    ]
    current-context = var.name
    kind            = "Config"
    preferences     = {}
    users = [
      {
        user = {
          client-certificate-data : base64encode(tls_locally_signed_cert.master_user.cert_pem)
          client-key-data : base64encode(tls_private_key.master_user.private_key_pem)
        }
        name : "master-user"
      }
    ]
  })

  filename        = "${path.root}/config/${var.name}.yaml"
  file_permission = "0700"
}

// note: hosts in this file need to be resolvable from the host running OpenTofu
output "kubeconfig" {
  value = abspath(local_file.kubeconfig.filename)
}

// note: must match the host in kubeconfig
output "local_kubernetes_api_url" {
  value = local.local_kubernetes_api_url
}

output "context" {
  value = var.name
}

output "ingress_class_name" {
  value = null
}
