variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "docker_io_proxy_directory" {
  description = "Directory in which to save docker.io's pull proxy registry data"
  default     = "/tmp/docker-io-registry"
}
