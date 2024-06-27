variable "name" {
  description = "Symbolic name of the host to SSH to"
  type        = string
}

variable "private_name" {
  description = "Private DNS name of the host to SSH to"
  type        = string
}

variable "public_name" {
  description = "Public DNS name of the host to SSH to (if any)"
  type        = string
  default     = null
}

variable "ssh_user" {
  description = "User name used for SSH access in the host and the bastion"
  default     = "root"
}

variable "ssh_private_key_path" {
  description = "Path of private SSH key for the host and the bastion"
  type        = string
}

variable "ssh_bastion_host" {
  description = "Public name of the SSH bastion host (if any)"
  type        = string
  default     = null
}

variable "ssh_tunnels" {
  description = "Opens SSH tunnels to localhost, optionally via the bastion. [[local_port, remote_port], ...]]"
  type        = list(list(number))
  default     = []
}
