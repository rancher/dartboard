module "host" {
  source                      = "../../${var.backend}/node"
  project_name                = var.project_name
  name                        = var.name
  ssh_private_key_path        = var.ssh_private_key_path
  ssh_user                    = var.ssh_user
  ssh_tunnels                 = var.ssh_tunnels
  host_configuration_commands = var.host_configuration_commands
  backend_variables           = var.backend_variables
  backend_network_variables   = var.network_backend_variables
  public                      = var.public
}
