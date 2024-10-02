variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this instance"
  type        = string
}

variable "default_image_id" {
  description = "ID of the VM image when image_name is not specified"
  type        = string
  default     = null
}

variable "image_name" {
  description = "Image name for this VM. Must be already present in Harvester. Requires image_namespace"
  type        = string
  default     = null
}

variable "image_namespace" {
  description = "Namespace for image_name. Must be already present in Harvester"
  type        = string
  default     = null
}

variable "cpu" {
  description = "Number of CPUs to allocate for the VM(s)"
  default     = 2
}

variable "memory" {
  description = "Number of GB of Memory to allocate for the VM(s)"
  default     = 8
}

variable "namespace" {
  description = "The namespace where the VM should be created"
  default     = "default"
}

variable "tags" {
  description = "A map of strings to add as VM tags"
  type        = map(string)
  default     = {}
}

variable "networks" {
  description = <<-EOT
  List of objects combining fields that define pre-existing VM Networks as well as the VM's network_interface type and model.
  Each object includes a name, a "public" flag if the network will assign a public IP address, a "wait_for_lease" flag if the interface is expected to provision an IP address,
  and optionally a namespace, interface_type and interface_model to be assigned to the VM.
  If using a VM Network which will assign a public IP to the VM, ensure the "public" flag is set to true.
  EOT
  type = list(object({
    name            = string
    namespace       = optional(string)
    interface_type  = optional(string, "bridge")
    interface_model = optional(string, "virtio")
    public          = bool
    wait_for_lease  = bool
  }))
  default = [{
    name           = "vmnet-shared"
    namespace      = "default"
    interace_type  = "bridge"
    public         = true
    wait_for_lease = true
  }]
}

variable "user" {
  description = "User name to use for VM access"
  type        = string
  default     = "opensuse"
}

variable "password" {
  description = "Password to use for VM access"
  type        = string
}

variable "ssh_public_key_id" {
  description = "ID of the public ssh key used to access the instance, see harvester_network"
  type        = string
}

variable "ssh_public_key" {
  description = "Contents of the public ssh key used to access the instance, see harvester_network"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access the instance"
  type        = string
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host. Leave null for publicly accessible instances"
  type        = string
  default     = null
}

variable "ssh_bastion_user" {
  description = "User name for the SSH bastion host's OS"
  default     = "root"
}

variable "ssh_bastion_key_path" {
  description = "Path of private ssh key used to access the bastion host"
  type        = string
  default     = null
}

variable "ssh_shared_public_keys" {
  description = "A list of shared public ssh key names + namespaces (which already exists in Harvester) to load onto the Harvester VMs"
  type = list(object({
    name      = string
    namespace = string
  }))
  default = []
}

variable "ssh_tunnels" {
  description = "Opens SSH tunnels to this host via the bastion"
  type        = list(list(number))
  default     = []
}

variable "disks" {
  description = "List of objects representing the disks to be provisioned for the VM. NOTE: boot_order will be set to the index of each disk in the list."
  type = list(object({
    name = string
    type = string
    size = number
    bus  = string
  }))
  default = [{
    name = "disk-0",
    type = "disk"
    size = 35
    bus = "virtio"
  }]
}

variable "efi" {
  description = "Flag that determines if the VM will boot in EFI mode"
  type        = bool
  default     = false
}

variable "secure_boot" {
  description = "Flag that determines if the VM will be provisioned with secure_boot enabled. EFI must be enabled to use this"
  type        = bool
  default     = false
}

variable "cloudinit_secrets" {
  description = <<-EOT
  A map which includes the name, namespace and optionally, the userdata content of a cloudinit configuration to be passed to the VM.
  If user_data is provided, a new cloudinit configuration will be created.
  If user_data is NOT provided, we use a datasource to pull the cloudinit_secret from Harvester.
  EOT
  type = list(object({
    name      = string
    namespace = string
    user_data = optional(string, "") //String content to be provided as user_data
  }))
  default = []
}

variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default     = [
    "echo 'Waiting for cloud-init to complete...'",
    "cloud-init status --wait > /dev/null",
    "echo 'Completed cloud-init!'",
    "cat /etc/os-release"
  ]
}
