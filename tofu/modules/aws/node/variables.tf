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
  default     = "ubuntu"
}

variable "ssh_tunnels" {
  description = "Opens SSH tunnels to this host via the bastion"
  type        = list(list(number))
  default     = []
}

variable "public" {
  description = "Whether the node is publicly accessible"
  default     = false
}

variable "node_module_variables" {
  description = <<EOT
    AWS-specific configuration variables.
    ami: AMI ID (see https://pint.suse.com/ to find others)
    instance_type: EC2 instance type
    root_volume_size_gb: Size of the root volume
    tags: A map of strings to add as instance tags
    host_configuration_commands: Commands to run when the host is deployed
  EOT
  type = object({
    ami : string,
    instance_type : string,
    root_volume_size_gb : number,
    host_configuration_commands: optional(list(string), ["cat /etc/os-release"])
    tags: optional(map(string)),
  })
  default = {
    ami : "ami-0e55a8b472a265e3f", // openSUSE-Leap-15-5-v20230608-hvm-ssd-arm64-a516e959-df54-4035-bb1a-63599b7a6df9
    instance_type : "t4g.small",
    root_volume_size_gb : 50,
    host_configuration_commands = ["cat /etc/os-release"]
    tags : {},
  }
}

variable "network_config" {
  description = <<EOT
    subnet_id: ID of the subnet to connect to
    vpc_security_group_id: ID of the security group to connect to
    ssh_key_name: Name of the SSH key used to access the host
    ssh_bastion_host: Public name of the SSH bastion host. null for publicly accessible instances
  EOT
  type = object({
    availability_zone : string,
    public_subnet_id : string,
    private_subnet_id : string,
    secondary_private_subnet_id : string,
    public_security_group_id : string,
    private_security_group_id : string,
    other_security_group_ids: optional(list(string), [""])
    ssh_key_name : string,
    ssh_bastion_host : string,
    ssh_bastion_user : string,
  })
}
