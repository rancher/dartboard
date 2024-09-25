# resource "aws_db_subnet_group" "subnet_group" {
#   name       = "${var.project_name}-${var.name}-db-subnet-group"
#   subnet_ids = [var.subnet_id, var.secondary_subnet_id]

#   tags = {
#     Project = var.project_name
#     Name    = "${var.project_name}-db-subnet"
#   }
# }

# resource "aws_db_parameter_group" "db_parameter_group_mariadb" {
#   name   = "${var.project_name}-${var.name}-db-parameter-group-mariadb"
#   family = "mariadb10.6"

#   parameter {
#     name  = "character_set_server"
#     value = "utf8"
#   }

#   parameter {
#     name  = "character_set_client"
#     value = "utf8"
#   }
# }

# resource "aws_db_parameter_group" "db_parameter_group_postgres" {
#   name   = "${var.project_name}-${var.name}-db-parameter-group-postgres"
#   family = "postgres14"
# }

# resource "aws_db_instance" "instance" {
#   depends_on = [
#     aws_db_parameter_group.db_parameter_group_mariadb, aws_db_parameter_group.db_parameter_group_postgres
#   ]
#   identifier        = "${var.project_name}-${var.name}"
#   instance_class    = var.instance_class
#   availability_zone = var.availability_zone

#   allocated_storage      = var.allocated_storage_gb
#   iops                   = var.iops
#   db_name                = var.name
#   engine                 = var.datastore
#   engine_version         = var.datastore == "mariadb" ? "10.6" : (var.datastore == "postgres" ? "14.5" : null)
#   username               = var.username
#   password               = var.password
#   parameter_group_name   = "${var.project_name}-${var.name}-db-parameter-group-${var.datastore}"
#   skip_final_snapshot    = true
#   db_subnet_group_name   = aws_db_subnet_group.subnet_group.name
#   vpc_security_group_ids = [var.vpc_security_group_id]

#   allow_major_version_upgrade = false
#   apply_immediately           = true
#   auto_minor_version_upgrade  = false
# }
