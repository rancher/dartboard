data "local_file" "rke_kubeconfig" {
  depends_on = [null_resource.rke_up_execution]
  count      = length(var.server_names) > 0 ? 1 : 0
  filename   = "${path.module}/../../config/rke_config/kube_config_${var.name}.yaml"
}

resource "local_file" "kubeconfig" {
  count = length(var.server_names) > 0 ? 1 : 0
  content = yamlencode({
    apiVersion = "v1"
    clusters = [{
      cluster = {
        certificate-authority-data = yamldecode(data.local_file.rke_kubeconfig[0].content)["clusters"][0]["cluster"]["certificate-authority-data"]
        server                     = "https://${var.sans[0]}:${var.ssh_local_port}"
      }
      name = var.sans[0]
    }]
    contexts = [{
      context = {
        cluster = var.sans[0]
        user : "kube-admin-local"
      }
      name = var.sans[0]
    }]
    current-context = var.sans[0]
    kind            = "Config"
    preferences     = {}
    users = [{
      user = {
        client-certificate-data : yamldecode(data.local_file.rke_kubeconfig[0].content)["users"][0]["user"]["client-certificate-data"]
        client-key-data : yamldecode(data.local_file.rke_kubeconfig[0].content)["users"][0]["user"]["client-key-data"]
      }
      name : "kube-admin-local"
    }]
  })

  filename = "${path.module}/../../config/${var.name}.yaml"
}
