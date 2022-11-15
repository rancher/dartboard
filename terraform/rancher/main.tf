resource "helm_release" "cert-manager" {
  name             = "cert-manager"
  chart            = var.cert_manager_chart
  namespace        = "cert-manager"
  create_namespace = true
  set {
    name  = "installCRDs"
    value = true
  }
}

resource "helm_release" "rancher" {
  depends_on       = [helm_release.cert-manager]
  name             = "rancher"
  chart            = var.chart
  namespace        = "cattle-system"
  create_namespace = true

  set {
    name  = "bootstrapPassword"
    value = "admin"
  }
  set {
    name  = "extraEnv[0].name"
    value = "CATTLE_SERVER_URL"
  }
  set {
    name  = "extraEnv[0].value"
    value = "https://${var.private_name}"
  }
  set {
    name  = "extraEnv[1].name"
    value = "CATTLE_BOOTSTRAP_PASSWORD"
  }
  set {
    name  = "extraEnv[1].value"
    value = "admin"
  }
  set {
    name  = "hostname"
    value = var.private_name
  }
  set {
    name  = "replicas"
    value = 1
  }
}

resource "helm_release" "rancher_configurator" {
  depends_on = [helm_release.rancher]
  name       = "rancher-configurator"
  chart      = "./rancher/rancher-configurator"

  set {
    name  = "publicName"
    value = var.public_name
  }
}
