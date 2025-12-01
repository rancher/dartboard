variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "name" {
  description = "Symbolic name of this cluster"
  type        = string
}

variable "distro_version" {
  description = "k3s version"
  default     = "v1.23.10+k3s1"
}

variable "server_count" {
  description = "Number of server nodes in this cluster"
  default     = 1
}

variable "agent_count" {
  description = "Number of agent nodes in this cluster"
  default     = 0
}

variable "reserve_node_for_monitoring" {
  description = "Whether to reserve a node for monitoring. If true, adds a taint and toleration with label 'monitoring' to the first agent node"
  default     = false
}

variable "ssh_private_key_path" {
  description = "Ignored"
  type        = string
  default     = null
}

variable "ssh_user" {
  description = "Ignored"
  type        = string
  default     = null
}

variable "local_kubernetes_api_port" {
  description = "Local port this cluster's Kubernetes API will be published to (via SSH tunnel)"
  default     = 6445
  type        = number
}

variable "tunnel_app_http_port" {
  description = "Local port this cluster's http endpoints will be published to (via SSH tunnel)"
  default     = 8080
  type        = number
}

variable "tunnel_app_https_port" {
  description = "Local port this cluster's https endpoints will be published to (via SSH tunnel)"
  default     = 8443
  type        = number
}

variable "sans" {
  description = "Additional Subject Alternative Names"
  type        = list(string)
  default     = []
}

variable "max_pods" {
  description = "Ignored"
  type        = number
  default     = null
}

variable "node_cidr_mask_size" {
  description = "Ignored"
  type        = number
  default     = null
}

variable "enable_audit_log" {
  description = "Enable Kubernetes API audit log to /var/log/k3d/audit for server nodes. Assumes a /var/lib/k3d/audit/audit.yaml file exists on the host"
  default     = false
}

variable "datastore_endpoint" {
  description = "Ignored"
  type        = string
  default     = null
}

variable "create_tunnels" {
  description = "Ignored (k3d publishes ports directly, no SSH tunnels needed)"
  type        = bool
  default     = false
}

variable "public" {
  description = "Ignored (k3d clusters are always local)"
  type        = bool
  default     = false
}

variable "node_module" {
  description = "Ignored"
  type        = string
  default     = null
}

variable "node_module_variables" {
  description = "Ignored"
  type        = any
  default     = null
}

variable "network_config" {
  description = "Network module outputs, to be passed to node_module"
  type        = any
}

variable "image" {
  description = "Set a k3s image, overriding k3s version"
  type        = string
  default     = null
}

variable "enable_metrics" {
  description = "Metrics are disabled by default due to https://github.com/kubernetes/kubernetes/issues/104459"
  default     = false
}

variable "first_metrics_port" {
  description = "Starting port to publish cluster nodes' metric endpoints"
  default     = 10250
}

variable "log_level" {
  description = "Change the logging level (up to 6 for trace)"
  type        = number
  default     = null
}

variable "datastore" {
  description = "Data store to use: mariadb, postgres, sqlite, embedded_etcd"
  default     = "embedded_etcd"
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
