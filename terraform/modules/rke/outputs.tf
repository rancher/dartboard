data "local_file" "rke_kubeconfig" {
  depends_on = [null_resource.rke_up_execution]
  count      = length(var.server_names) > 0 ? 1 : 0
  filename   = "${path.root}/config/rke_config/kube_config_${var.name}.yaml"
}

resource "local_file" "kubeconfig" {
  count = length(var.server_names) > 0 ? 1 : 0
  content = yamlencode({
    apiVersion = "v1"
    clusters = [
      {
        cluster = {
          certificate-authority-data = yamldecode(data.local_file.rke_kubeconfig[0].content)["clusters"][0]["cluster"]["certificate-authority-data"]
          server                     = "https://${var.sans[0]}:${var.local_kubernetes_api_port}"
        }
        name = var.name
      }
    ]
    contexts = [
      {
        context = {
          cluster = var.name
          user : "kube-admin-local"
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
          client-certificate-data : yamldecode(data.local_file.rke_kubeconfig[0].content)["users"][0]["user"]["client-certificate-data"]
          client-key-data : yamldecode(data.local_file.rke_kubeconfig[0].content)["users"][0]["user"]["client-key-data"]
        }
        name : "kube-admin-local"
      }
    ]
  })

  filename        = "${path.root}/config/${var.name}.yaml"
  file_permission = "0700"
}

output "kubeconfig" {
  value = length(var.server_names) > 0 ? abspath(local_file.kubeconfig[0].filename) : null
}

output "context" {
  value = var.name
}

output "ingress_class_name" {
  value = null
}
