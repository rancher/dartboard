resource "local_file" "kubeconfig" {
  content = yamlencode({
    apiVersion = "v1"
    clusters = [{
      cluster = {
        certificate-authority-data = base64encode(var.server_ca_cert)
        server                     = "https://${var.sans[0]}:${var.ssh_local_port}"
      }
      name = var.sans[0]
    }]
    contexts = [{
      context = {
        cluster = var.sans[0]
        user : "master-user"
      }
      name = var.sans[0]
    }]
    current-context = var.sans[0]
    kind            = "Config"
    preferences     = {}
    users = [{
      user = {
        client-certificate-data : base64encode(var.master_user_cert)
        client-key-data : base64encode(var.master_user_key)
      }
      name : "master-user"
    }]
  })

  filename = "${path.module}/../config/${var.name}.yaml"
}
