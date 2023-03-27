variable "private_name" {
  description = "Name of the Rancher Server (API), as seen from downstream clusters"
  type        = string
}

variable "private_port" {
  description = "Port of the Rancher Server (API), as seen from downstream clusters"
  default = 443
}

variable "public_name" {
  description = "Name of the Rancher Server (API), as seen from the host running kubectl"
  type        = string
}

variable "bootstrap_password" {
  description = "Bootstrap password for this Rancher server"
  default = "admin"
}

variable "cert_manager_chart" {
  default = "https://charts.jetstack.io/charts/cert-manager-v1.8.0.tgz"
}

variable "chart" {
  default = "https://releases.rancher.com/server-charts/latest/rancher-2.6.9.tgz"
}
