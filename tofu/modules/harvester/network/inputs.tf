variable "network_name" {
  type = string
}

variable "clusternetwork_name" {
  type = string
}

variable "namespace" {
  description = "The namespace for hosts created by this module"
  default     = "default"
}

variable "ssh_public_key_path" {
  description = "Path of public ssh key for hosts created by this module"
  type        = string
}

variable "create" {
  type = bool
  default = false
}

variable "vlan_uplink" {
  description = "Harvester ClusterNetwork uplink configuration"
  type = object({
    nics = list(string)
    bond_miimon = optional(number)
    bond_mode = optional(string)
    mtu = optional(number)
  })
  default = null
}

variable "network_config" {
  type = object({
    vlan_id = number
    route_mode = string
    route_dhcp_server_ip = string
    route_cidr = string
    route_gateway = string
  })
  default = null
}
