variable "ssh_public_key_path" {
  description = "Path to SSH public key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519.pub"
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519"
}

variable "ssh_user" {
  description = "User name to use for the SSH connection to all nodes in all clusters"
  default     = "opensuse"
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

variable "upstream_cluster" {
  description = "Upstream cluster configuration. See tofu/modules/generic/test_environment/variables.tf for details"
  type        = any
}

variable "upstream_cluster_distro_module" {
  description = "Name of the module to use for the upstream cluster"
  default     = "generic/k3s"
}

variable "downstream_cluster_templates" {
  description = "List of downstream cluster configurations. See tofu/modules/generic/test_environment/variables.tf for details"
  type        = list(any)
}

variable "downstream_cluster_distro_module" {
  description = "Name of the module to use for the downstream clusters"
  default     = "generic/k3s"
}

variable "node_templates" {
  description = "List of node configurations. See tofu/modules/generic/test_environment/variables.tf for details"
  type        = list(any)
}

variable "tester_cluster" {
  description = "Tester cluster configuration. See tofu/modules/generic/test_environment/variables.tf for details"
  type        = any
  default     = null
}

variable "tester_cluster_distro_module" {
  description = "Name of the module to use for the downstream clusters"
  default     = "generic/k3s"
}

# "Multi-tenancy" variables
variable "project_name" {
  description = "Name of this project, used as prefix for resources it creates"
  default     = "st"
}

variable "first_kubernetes_api_port" {
  description = "Port number where the Kubernetes API of the first cluster is published locally. Other clusters' ports are published in successive ports"
  type        = number
  default     = 7445
}

variable "first_app_http_port" {
  description = "Port number where the first server's port 80 is published locally. Other clusters' ports are published in successive ports"
  type        = number
  default     = 9080
}

variable "first_app_https_port" {
  description = "Port number where the first server's port 443 is published locally. Other clusters' ports are published in successive ports"
  type        = number
  default     = 9443
}

# Harvester-specific variables
variable "namespace" {
  description = "The namespace where the VMs should be created"
  default     = "default"
}

variable "kubeconfig" {
  description = "Path to the Harvester kubeconfig file. Uses KUBECONFIG by default. See https://docs.harvesterhci.io/v1.3/faq/#how-can-i-access-the-kubeconfig-file-of-the-harvester-cluster"
  type        = string
  nullable    = false
}

variable "network" {
  description = <<-EOT
  An object combining fields that define a pre-existing VM Network as well as the VM's network_interface type and model.
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

variable "password" {
  description = "Password to use for VM access (via terminal, SSH access is exclusively via SSH public key)"
  default     = "linux"
}

variable "ssh_shared_public_keys" {
  description = "A list of shared public ssh key names + namespaces (which already exists in Harvester) to load onto the Harvester VMs"
  type = list(object({
    name      = string
    namespace = string
  }))
  default = []
}

variable "create_image" {
  description = "Whether to create a new image for the VMs"
  default     = true
}
