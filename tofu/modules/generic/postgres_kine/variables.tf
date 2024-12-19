variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "ssh_user" {
  description = "User name to use for the SSH connection to the host"
  type        = string
  default     = "root"
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access cluster nodes"
  type        = string
}

variable "gogc" {
  description = "Tunable parameter for Go's garbage collection, see: https://tip.golang.org/doc/gc-guide"
  type        = number
  default     = null
}

variable "kine_version" {
  description = "Kine version"
  default     = "v0.9.8"
}

variable "kine_executable" {
  description = "Overrides kine_version by copying an executable from this path"
  type        = string
  default     = null
}

variable "node_module" {
  description = "Non-generic module to create this node"
  type        = string
}

variable "node_backend_variables" {
  description = "Backend-specific configuration variables for the database host"
  type = any
}

variable "network_backend_variables" {
  description = "Backend-specific configuration variables for the database host"
  type = any
}
