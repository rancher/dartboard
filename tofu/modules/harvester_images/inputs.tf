variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "namespace" {
  description = "The namespace for hosts created by this module"
  default     = "default"
}

variable "create" {
  description = "Would create resources within the module"
  default     = true
}
