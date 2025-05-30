variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this host"
  type        = string
}

variable "ssh_private_key_path" {
  description = "Path of private ssh key used to access the host"
  type        = string
}

variable "ssh_user" {
  description = "User name to use for the SSH connection to the host"
  default     = "root"
}

variable "ssh_tunnels" {
  description = "Opens SSH tunnels to this host via the bastion"
  type        = list(list(number))
  default     = []
}

variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default = [
    "echo 'Waiting for cloud-init to complete...'",
    "cloud-init status --wait > /dev/null",
    "echo 'Completed cloud-init!'",
    "cat /etc/os-release"
  ]
}

variable "public" {
  description = "Whether the node is publicly accessible"
  default     = false
}

variable "node_module_variables" {
  description = <<EOT
    Harvester-specific VM configuration variables.
    image_name: Image name for this VM. If null, the image created by the network module will be used
    image_namespace: Namespace for image_name. If null, the image created by the network module will be used
    image_id: Image ID as an alternative to name and namespace. If null, the image created by the network module will be used
    cpu: Number of CPUs to allocate for the VM(s)
    memory: Number of GB of Memory to allocate for the VM(s)
    tags: A map of strings to add as VM tags
    password: Password to use for VM access (via terminal, SSH access is exclusively via SSH public key)
    ssh_shared_public_keys: A list of shared public ssh key names + namespaces (which already exists in Harvester) to load onto the Harvester VMs
    disks: List of objects representing the disks to be provisioned for the VM. NOTE: boot_order will be set to the index of each disk in the list.
    efi: Flag that determines if the VM will boot in EFI mode
    secure_boot: Flag that determines if the VM will be provisioned with secure_boot enabled. EFI must be enabled to use this
  EOT
  type = object({
    image_name      = optional(string)
    image_namespace = optional(string)
    image_id        = optional(string)
    cpu             = number
    memory          = number
    tags            = optional(map(string))
    password        = string
    ssh_shared_public_keys = optional(list(object({
      name      = string
      namespace = string
    })))
    disks = optional(list(object({
      name = string
      type = string
      size = number
      bus  = string
    })))
    efi         = optional(bool)
    secure_boot = optional(bool)
  })
  default = {
    image_name             = null
    image_namespace        = null
    cpu                    = 2
    memory                 = 8
    namespace              = "default"
    tags                   = {}
    password               = "linux"
    ssh_shared_public_keys = []
    disks = [{
      name = "disk-0"
      type = "disk"
      size = 35
      bus  = "virtio"
    }]
    efi         = false
    secure_boot = false
  }
}

variable "network_config" {
  description = <<EOT
    Harvester-specific network configuration variables.
    namespace: The namespace for nodes created by this module
    ssh_public_key_id: ID of the public ssh key used to access the instance
    ssh_public_key: Contents of the public ssh key used to access the instance
    id: ID of the network
    name: Name of the network
    clusternetwork_name: Name of the cluster network
    interface_type: Type of network interface to use
    interface_model: Model of network interface to use
    public: Whether the network will assign a public IP address
    wait_for_lease: Whether the interface is expected to provision an IP address
    created_image_id: ID of the image if one was created
    ssh_bastion_host: Public name of the SSH bastion host. Leave null for publicly accessible instances
    ssh_bastion_user: User name for the SSH bastion host's OS
    ssh_bastion_key_path: Path of private ssh key used to access the bastion host
  EOT
  type = object({
    namespace            = string
    ssh_public_key_id    = string
    ssh_public_key       = string
    id                   = string
    name                 = string
    clusternetwork_name  = string
    interface_type       = string
    interface_model      = string
    public               = bool
    wait_for_lease       = bool
    created_image_id     = optional(string)
    ssh_bastion_host     = optional(string)
    ssh_bastion_user     = optional(string)
    ssh_bastion_key_path = optional(string)
  })
}
