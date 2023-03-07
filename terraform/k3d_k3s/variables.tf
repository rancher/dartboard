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
  description = "Name of the Docker network to connect containers to (or null)"
  type        = string
  default     = null
}

variable "kubernetes_api_port" {
  description = "Port k3s cluster's Kubernetes API will be published to"
  default     = 6445
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
  description = "Data store to use: mariadb, postgres or default for an automatic choice (sqlite for one-server-node installs, embedded etcd otherwise)"
  default     = "default"
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
  description = "Enable pprof endpoint on supervisor port. Beware: this breaks cert-manager until https://github.com/k3s-io/k3s/pull/6635 is merged"
  default     = false
}

variable "gogc" {
  description = "Tunable parameter for Go's garbage collection, see: https://tip.golang.org/doc/gc-guide"
  type        = number
  default     = null
}

variable "postgres_log_min_duration_statement" {
  description = "Set to log all statements taking longer than the specified amount of milliseconds, https://www.postgresql.org/docs/15/runtime-config-logging.html#GUC-LOG-MIN-DURATION-STATEMENT"
  type        = number
  default     = null
}

variable "kine_image" {
  description = "Kine container image"
  default     = "rancher/kine:v0.9.8"
}

variable "kine_debug" {
  description = "Set to true to enable kine debug logging"
  default     = false
}
