
variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "namespace" {
  description = "The namespace for hosts created by this module"
  default     = "default"
}


variable "network_details" {
  description = <<-EOT
  An object combining fields that define a VM Network as well as the VM's network_interface type and model.
  The object includes a name, a "public" flag if the network will assign a public IP address, a "wait_for_lease" flag if the interface is expected to provision an IP address,
  and optionally a namespace, interface_type and interface_model to be assigned to the VM.
  If using a VM Network which will assign a public IP to the VM, ensure the "public" flag is set to true.
  EOT
  type = object({
    create              = bool
    name                = string
    vlan_id             = number
    clusternetwork_name = string
    namespace           = optional(string)
    interface_type      = optional(string)
    interface_model     = optional(string)
    public              = bool
    wait_for_lease      = bool
  })
  default = {
    create              = false
    clusternetwork_name = "vmnet"
    vlan_id             = 100
    name                = "vmnet-shared"
    namespace           = "default"
    interace_type       = "bridge"
    public              = true
    wait_for_lease      = true
  }
}

variable "create_image" {
  description = "Would create resources within the module"
  default     = true
}

variable "ssh_public_key_path" {
  description = "Path of public ssh key for hosts created by this module"
  type        = string
}

variable "vlan_uplink" {
  description = "Harvester ClusterNetwork uplink configuration"
  type = object({
    nics        = list(string)
    bond_miimon = optional(number)
    bond_mode   = optional(string)
    mtu         = optional(number)
  })
  default = null
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible Harvester instances"
  type        = string
  default     = null
}

variable "ssh_bastion_user" {
  description = "User name to connect to the SSH bastion host"
  default     = null
}

variable "ssh_bastion_key_path" {
  description = "Path of private ssh key used to access the bastion host to access Harvester"
  type        = string
  default     = null
}
