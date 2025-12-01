variable "project_name" {
  description = "A prefix for names of objects created by this module"
  type        = string
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
  type        = string
  default     = "db.t4g.xlarge"
}

variable "iops" {
  description = "The amount of provisioned IOPS"
  default     = null
  type        = number
}

variable "subnet_id" {
  description = "ID of the subnet to connect to"
  type        = string
}

variable "secondary_subnet_id" {
  description = "ID of the secondary subnet to connect to"
  type        = string
}

variable "vpc_security_group_id" {
  description = "ID of the security group to connect to"
  type        = string
}

variable "datastore" {
  description = "Data store to use: mariadb, postgres or leave for a default (sqlite for one-server-node installs, embedded etcd otherwise)"
  type        = string
  default     = null
}

variable "allocated_storage_gb" {
  description = "Size of DB allocated storage"
  type        = number
  default     = 20
}

variable "username" {
  description = "The database's main user name"
  type        = string
  default     = "rdsuser"
}

variable "password" {
  description = "The database's main user password"
  type        = string
  default     = "v3ryverysecre7"
}
