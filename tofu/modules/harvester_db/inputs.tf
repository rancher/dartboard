variable "project_name" {
  description = "A prefix for names of objects created by this module"
  default     = "st"
}

variable "availability_zone" {
  description = "Availability zone where the instance is created"
  type        = string
}

variable "name" {
  description = "Symbolic name of this instance"
  type        = string
}

variable "instance_class" {
  description = "RDS instance class"
  default     = "db.t4g.xlarge"
}

variable "iops" {
  description = "The amount of provisioned IOPS"
  default     = null
  type        = number
}

variable "datastore" {
  description = "Data store to use: mariadb, postgres or leave for a default (sqlite for one-server-node installs, embedded etcd otherwise)"
  default     = null
}

variable "allocated_storage_gb" {
  description = "Size of DB allocated storage"
  default     = 20
}

variable "username" {
  description = "The database's main user name"
  default     = "rdsuser"
}

variable "password" {
  description = "The database's main user password"
  default     = "v3ryverysecre7"
}
