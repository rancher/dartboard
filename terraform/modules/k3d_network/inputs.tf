variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "first_proxy_port" {
  description = "Port to publish k3d's internal registry for registry-1.docker.io"
  default     = 5001
}

variable "registry_pull_proxies" {
  description = "URLs of registries to create k3d pull proxies for"
  default = [
    {
      name = "docker.io"
      url  = "https://registry-1.docker.io"
    },
    {
      name = "quay.io"
      url  = "https://quay.io"
    },
    {
      name = "registry.suse.com"
      url  = "https://registry.suse.com"
    },
  ]
}
