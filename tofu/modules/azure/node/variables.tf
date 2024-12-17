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
  default     = "azureuser"
}

variable "ssh_tunnels" {
  description = "Opens SSH tunnels to this host via the bastion"
  type        = list(list(number))
  default     = []
}

variable "host_configuration_commands" {
  description = "Commands to run when the host is deployed"
  default     = ["cat /etc/os-release"]
}

variable "public" {
  description = "Whether the node is publicly accessible"
  default     = false
}

variable "backend_variables" {
  description = <<EOT
    Azure-specific configuration variables.
    os_image: Azure VM OS image
    size: Azure VM kind
    is_spot: Whether to use spot instances
    os_disk_type: Azure VM OS disk type
    os_disk_size: Azure VM OS disk size
    os_ephemeral_disk: Whether to use ephemeral disk for the OS
  EOT
  type = object({
    os_image : object({
      publisher : string
      offer : string
      sku : string
      version : string
    })
    size : string
    is_spot : bool
    os_disk_type : string
    os_disk_size : number
    os_ephemeral_disk : bool
  })
  default = {
    os_image = {
      publisher = "SUSE"
      offer     = "openSUSE-Leap"
      sku       = "15-2"
      version   = "latest"
    }
    size              = "Standard_B2as_v2"
    is_spot           = false
    os_disk_type      = "Premium_LRS"
    os_disk_size      = 30
    os_ephemeral_disk = false
  }
}

variable "network_backend_variables" {
  description = <<EOT
    location: Azure location
    resource_group_name: Azure resource group name
    public_subnet_id: Azure public subnet id
    private_subnet_id: Azure private subnet id
    ssh_public_key_path: Path to the SSH public key
    ssh_bastion_host: Public name of the SSH bastion host
    ssh_bastion_user: User name for the SSH bastion host's OS
    storage_account_uri: Storage account URI to attach to the VM to enable Boot Diagnostics
  EOT
  type = object({
    location : string,
    resource_group_name : string,
    public_subnet_id : string,
    private_subnet_id : string,
    ssh_public_key_path : string,
    ssh_bastion_host : string,
    ssh_bastion_user : string,
    storage_account_uri : string,
  })
}
