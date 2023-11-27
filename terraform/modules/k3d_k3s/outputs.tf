resource "local_file" "kubeconfig" {
  count = var.server_count > 0 ? 1 : 0
  content = yamlencode({
    apiVersion = "v1"
    clusters = [
      {
        cluster = {
          certificate-authority-data = base64encode(k3d_cluster.cluster[0].credentials[0].cluster_ca_certificate)
          server                     = k3d_cluster.cluster[0].credentials[0].host
        }
        name = "k3d-${var.project_name}-${var.name}"
      }
    ]
    contexts = [
      {
        context = {
          cluster = "k3d-${var.project_name}-${var.name}"
          user : "admin@k3d-${var.project_name}-${var.name}"
        }
        name = "k3d-${var.project_name}-${var.name}"
      }
    ]
    current-context = "k3d-${var.project_name}-${var.name}"
    kind            = "Config"
    preferences     = {}
    users = [
      {
        user = {
          client-certificate-data : base64encode(k3d_cluster.cluster[0].credentials[0].client_certificate)
          client-key-data : base64encode(k3d_cluster.cluster[0].credentials[0].client_key)
        }
        name : "admin@k3d-${var.project_name}-${var.name}"
      }
    ]
  })

  filename        = "${path.root}/config/${var.name}.yaml"
  file_permission = "0700"
}

output "kubeconfig" {
  value = var.server_count > 0 ? abspath(local_file.kubeconfig[0].filename) : null
}

output "context" {
  value = "k3d-${var.project_name}-${var.name}"
}

output "first_server_private_name" {
  value = "k3d-${var.project_name}-${var.name}-server-0"
}

output "first_server_public_name" {
  value = null
}

output "local_http_port" {
  value = var.local_http_port
}

output "local_https_port" {
  value = var.local_https_port
}

output "ingress_class_name" {
  value = null
}
