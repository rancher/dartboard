variable "network_name" {
  type = string
}

variable "vlanconfig_name" {
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

variable "vlan_id" {
  type = number
  default = null
}

variable "route_mode" {
  type = string
  default = null
}

variable "route_dhcp_server_ip" {
  type = string
  default = null
}

variable "route_cidr" {
  type = string
  default = null
}

variable "route_gateway" {
  type = string
  default = null
}

