variable "private_name" {
  description = "Name of the Rancher Server (API), as seen from downstream clusters"
  type = string
}

variable "public_name" {
  description = "Name of the Rancher Server (API), as seen from the host running kubectl"
  type = string
}

variable "cert_manager_chart" {
  default = "https://charts.jetstack.io/charts/cert-manager-v1.8.0.tgz"
}

variable "chart" {
  default = "https://releases.rancher.com/server-charts/latest/rancher-2.6.5.tgz"
}

variable "api_token_string" {
  description = "Pre-shared API token string to register downstream clusters"
  type = string
}