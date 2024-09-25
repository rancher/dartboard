variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "namespace" {
  description = "The namespace for hosts created by this module"
  default     = "default"
}

variable "ssh_public_key_path" {
  description = "Path of public ssh key for hosts created by this module"
  type        = string
}
