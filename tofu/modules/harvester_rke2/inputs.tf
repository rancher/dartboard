variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this instance"
  type        = string
}

variable "image_name" {
  description = "Image name for all VMs in this cluster"
  type        = string
}

variable "image_namespace" {
  description = "Namespace to search for OR upload image, if it does not exist"
  default     = "default"
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
  default = []
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

variable "ssh_keys" {
  description = "List of SSH key names and namespaces to be pulled from Harvester"
  type = list(object({
    name      = string
    namespace = string
  }))
  default = []
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

variable "ssh_tunnels" {
  description = "Opens SSH tunnels to this host via the bastion"
  type        = list(list(number))
  default     = []
}

variable "disks" {
  description = "List of objects representing the disks to be provisioned for the VM"
  type = list(object({
    name = string
    type = string
    size = number
    bus  = string
  }))
  default = []
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
    user_data = optional(string, "")
  }))
  default = []
}

variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default     = ["cat /etc/os-release"]
}

variable "server_count" {
  description = "Number of server nodes in this cluster"
  default     = 1
}

variable "agent_count" {
  description = "Number of agent nodes in this cluster"
  default     = 0
}

variable "agent_labels" {
  description = "Per-agent-node lists of labels to apply"
  type        = list(list(object({ key : string, value : string })))
  default     = []
}

variable "agent_taints" {
  description = "Per-agent-node lists of taints to apply"
  type        = list(list(object({ key : string, value : string, effect : string })))
  default     = []
}

variable "local_kubernetes_api_port" {
  description = "Local port this cluster's Kubernetes API will be published to (via SSH tunnel)"
  default     = 6445
}

variable "tunnel_app_http_port" {
  description = "Local port this cluster's http endpoints will be published to (via SSH tunnel)"
  default     = 8080
}

variable "tunnel_app_https_port" {
  description = "Local port this cluster's https endpoints will be published to (via SSH tunnel)"
  default     = 8443
}

variable "sans" {
  description = "Additional Subject Alternative Names"
  type        = list(string)
  default     = []
}

variable "distro_version" {
  description = "RKE2 version"
  default     = "v1.24.4+rke2r1"
}

variable "max_pods" {
  description = "Maximum number of pods per node"
  default     = 110
}

variable "node_cidr_mask_size" {
  description = "Size of the CIDR mask for nodes. Increase when increasing max_pods so that 2^(32-node_cidr_max_size) > 2 * max_pods"
  default     = 24
}
