variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "server_count" {
  description = "Number of server nodes in this cluster"
  default     = 1
}

variable "agent_count" {
  description = "Number of agent nodes in this cluster"
  default     = 0
}

variable "sans" {
  description = "Additional Subject Alternative Names"
  type        = list(string)
  default     = []
}

variable "distro_version" {
  description = "k3s version"
  default     = "v1.23.10+k3s1"
}

variable "image" {
  description = "Set a k3s image, overriding k3s version"
  type        = string
  default     = null
}

variable "network_name" {
  description = "Name of the Docker network to connect containers to"
  type        = string
}

variable "additional_port_mappings" {
  description = "Opens additional port mappings to the first server node (format is [[host_port, container_port]])"
  type        = list(list(number))
  default     = []
}

variable "log_level" {
  description = "Change the logging level (up to 6 for trace)"
  type        = number
  default     = null
}

variable "datastore" {
  description = "Data store to use: mariadb, postgres or leave for a default (sqlite for one-server-node installs, embedded etcd otherwise)"
  default     = null
}

variable "datastore_dbname" {
  description = "The database's name"
  default     = "kine"
}

variable "datastore_username" {
  description = "The database's main user name"
  default     = "kineuser"
}

variable "datastore_password" {
  description = "The database's main user password"
  default     = "kinepassword"
}

variable "enable_pprof" {
  description = "Enable pprof endpoint on supervisor port"
  default     = false
}

variable "gogc" {
  description = "Tunable parameter for Go's garbage collection, see: https://tip.golang.org/doc/gc-guide"
  type        = number
  default     = null
}
