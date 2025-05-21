module "host" {
  source                      = "../../${var.node_module}"
  project_name                = var.project_name
  name                        = var.name
  ssh_private_key_path        = var.ssh_private_key_path
  ssh_user                    = var.ssh_user
  ssh_tunnels                 = var.ssh_tunnels
  host_configuration_commands = var.host_configuration_commands
  node_module_variables       = var.node_module_variables
  network_config              = var.network_config
  public                      = var.public
}

resource "local_file" "ssh_script" {
  content = <<-EOT
    #!/bin/sh
    ssh -o "StrictHostKeyChecking=no" -o "UserKnownHostsFile=/dev/null" \
      -i ${var.ssh_private_key_path} \
      %{if var.network_config.ssh_bastion_host != null~}
      -o ProxyCommand="ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null -i ${var.ssh_private_key_path} -W %h:%p ${var.network_config.ssh_bastion_user}@${var.network_config.ssh_bastion_host}" ${var.ssh_user}@${module.host.private_name} \
      %{else~}
      ${var.ssh_user}@${module.host.public_name} \
      %{endif~}
      $@
  EOT

  filename = "${path.root}/${terraform.workspace}_config/ssh-to-${var.name}.sh"
}

resource "local_file" "open_tunnels" {
  count = length(var.ssh_tunnels) > 0 ? 1 : 0
  content = templatefile("${path.module}/open-tunnels-to.sh", {
    ssh_bastion_host     = var.network_config.ssh_bastion_host
    ssh_bastion_user     = var.network_config.ssh_bastion_user
    ssh_tunnels          = var.ssh_tunnels
    private_name         = module.host.private_name
    public_name          = module.host.public_name
    ssh_user             = var.ssh_user
    ssh_private_key_path = var.ssh_private_key_path
  })

  filename = "${path.root}/${terraform.workspace}_config/open-tunnels-to-${var.name}.sh"
}

resource "null_resource" "open_tunnels" {
  count = length(var.ssh_tunnels) > 0 ? 1 : 0
  provisioner "local-exec" {
    interpreter = ["bash", "-c"]
    command     = local_file.open_tunnels[0].filename
  }
  triggers = {
    always_run = timestamp()
  }
}
