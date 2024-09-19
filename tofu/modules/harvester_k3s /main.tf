module "server_nodes" {
  count                 = var.server_count
  source                = "../harvester_host"
  image_name            = var.image_name
  image_namespace       = var.image_namespace
  cpu                   = var.cpu
  memory                = var.memory
  project_name          = var.project_name
  name                  = "${var.name}-server-${count.index}"
  ssh_keys              = var.ssh_keys
  ssh_private_key_path  = var.ssh_private_key_path
  ssh_user              = var.ssh_user
  subnet_id             = var.subnet_id
  vpc_security_group_id = var.vpc_security_group_id
  ssh_bastion_host      = var.ssh_bastion_host
  ssh_bastion_user      = var.ssh_bastion_user
  ssh_tunnels = count.index == 0 ? [
    [var.local_kubernetes_api_port, 6443],
    [var.tunnel_app_http_port, 80],
    [var.tunnel_app_https_port, 443],
  ] : []
  host_configuration_commands = var.host_configuration_commands
}

module "agent_nodes" {
  count                       = var.agent_count
  source                      = "../harvester_host"
  image_name                  = var.image_name
  image_namespace             = var.image_namespace
  cpu                         = var.cpu
  memory                      = var.memory
  project_name                = var.project_name
  name                        = "${var.name}-agent-${count.index}"
  ssh_key_name                = var.ssh_key_name
  ssh_private_key_path        = var.ssh_private_key_path
  ssh_user                    = var.ssh_user
  subnet_id                   = var.subnet_id
  vpc_security_group_id       = var.vpc_security_group_id
  ssh_bastion_host            = var.ssh_bastion_host
  ssh_bastion_user            = var.ssh_bastion_user
  host_configuration_commands = var.host_configuration_commands
}

module "rds" {
  source                = "../harvester_db"
  count                 = var.datastore == null ? 0 : 1
  datastore             = var.datastore
  availability_zone     = var.availability_zone
  project_name          = var.project_name
  name                  = "kine"
  subnet_id             = var.subnet_id
  secondary_subnet_id   = var.secondary_subnet_id
  vpc_security_group_id = var.vpc_security_group_id
}

module "k3s" {
  source       = "../k3s"
  project      = var.project_name
  name         = var.name
  server_names = [for node in module.server_nodes : node.private_name]
  agent_names  = [for node in module.agent_nodes : node.private_name]
  agent_labels = var.agent_labels
  agent_taints = var.agent_taints
  sans         = compact(concat(var.sans, var.server_count > 0 ? [module.server_nodes[0].private_name, module.server_nodes[0].public_name] : []))

  ssh_private_key_path      = var.ssh_private_key_path
  ssh_user                  = var.ssh_user
  ssh_bastion_host          = var.ssh_bastion_host
  ssh_bastion_user          = var.ssh_bastion_user
  local_kubernetes_api_port = var.local_kubernetes_api_port

  distro_version      = var.distro_version
  max_pods            = var.max_pods
  node_cidr_mask_size = var.node_cidr_mask_size
  datastore_endpoint = (
    var.datastore_endpoint != null ?
    var.datastore_endpoint :
    var.datastore == "mariadb" ?
    "mysql://${module.rds[0].username}:${module.rds[0].password}@tcp(${module.rds[0].endpoint})/${module.rds[0].db_name}" :
    var.datastore == "postgres" ?
    "postgres://${module.rds[0].username}:${module.rds[0].password}@${module.rds[0].endpoint}/${module.rds[0].db_name}" :
    null
  )
}
