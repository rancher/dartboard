# Stub variables for k3d node module
# These are ignored as k3d doesn't support external nodes

variable "project_name" {
  description = "Ignored for k3d"
  default     = null
}

variable "name" {
  description = "Ignored for k3d"
  type        = string
  default     = ""
}

variable "ssh_private_key_path" {
  description = "Ignored for k3d"
  type        = string
  default     = ""
}

variable "ssh_user" {
  description = "Ignored for k3d"
  type        = string
  default     = ""
}

variable "ssh_tunnels" {
  description = "Ignored for k3d"
  type        = list(list(number))
  default     = []
}

variable "node_module_variables" {
  description = "Ignored for k3d"
  type        = any
  default     = null
}

variable "network_config" {
  description = "Ignored for k3d"
  type        = any
  default     = null
}

variable "public" {
  description = "Ignored for k3d"
  default     = false
}
