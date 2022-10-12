terraform {
  required_providers {
    ssh = {
      source = "loafoe/ssh"
    }
  }
}

resource "ssh_resource" "node_preparation" {
  count        = length(var.server_names) + length(var.agent_names)
  host         = concat(var.server_names, var.agent_names)[count.index]
  private_key  = file(var.ssh_private_key_path)
  user         = "root"
  bastion_host = var.ssh_bastion_host
  timeout      = "600s"

  file {
    content     = file("${path.module}/prepare_for_rke.sh")
    destination = "/root/prepare_for_rke.sh"
    permissions = "0700"
  }

  commands = [
    "/root/prepare_for_rke.sh > >(tee prepare_for_rke.log) 2> >(tee prepare_for_rke.err >&2)",
  ]
}

resource "null_resource" "rke_executable" {
  provisioner "local-exec" {
    interpreter = ["bash", "-c"]
    command = templatefile("${path.module}/download_rke.sh", {
      version     = var.rke_version
      os_platform = var.rke_os_platform
      target      = "${path.module}/../config"
    })
  }
}

resource "local_file" "rke_config" {
  content = templatefile("${path.module}/cluster.yaml", {
    server_names         = var.server_names
    agent_names          = var.agent_names
    ssh_private_key_path = var.ssh_private_key_path
    ssh_bastion_host     = var.ssh_bastion_host
    kubernetes_version   = var.kubernetes_version
    max_pods             = var.max_pods
    node_cidr_mask_size  = var.node_cidr_mask_size
    sans                 = var.sans
  })

  filename = "${path.module}/../config/rke_config/${var.name}.yaml"
}

resource "null_resource" "rke" {
  count      = length(var.server_names) > 0 ? 1 : 0
  depends_on = [ssh_resource.node_preparation, local_file.rke_config, null_resource.rke_executable]
  provisioner "local-exec" {
    command = "${path.module}/../config/rke up --config ${path.module}/../config/rke_config/${var.name}.yaml"
  }

  triggers = {
    node_names = join(",", concat(var.server_names, var.agent_names))
  }
}
