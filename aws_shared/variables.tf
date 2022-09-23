variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "ssh_public_key_path" {
  description = "Path of public ssh key for AWS"
  type = string
}