terraform {
  required_providers {
    ssh = {
      source  = "loafoe/ssh"
      version = "2.7.0"
    }
  }
}

locals {
  // Fall back to the node key when the network module does not define a
  // dedicated bastion key, so providers without one keep working.
  bastion_private_key = try(var.network_config.ssh_bastion_key_path, null) != null ? file(var.network_config.ssh_bastion_key_path) : file(var.ssh_private_key_path)

  # renovate: datasource=github-releases depName=etcd-io/etcd
  etcd_3_7_version = "v3.7.2"
  # renovate: datasource=github-releases depName=etcd-io/etcd digestVersion=v3.7.2
  etcd_3_7_sha256 = "3a3679bc51a4ee9d30bccea1da7cd4fe62c6fc1d2ca1255068d2c53bf3026135"

  # renovate: datasource=github-releases depName=etcd-io/etcd
  etcd_3_6_version = "v3.6.15"
  # renovate: datasource=github-releases depName=etcd-io/etcd digestVersion=v3.6.15
  etcd_3_6_sha256 = "b51d86e6bff2168de5ba8cd631c0fe6c785c7a1b32b083079f5677d6b7e2772c"

  # renovate: datasource=github-releases depName=etcd-io/etcd
  etcd_3_5_version = "v3.5.34"
  # renovate: datasource=github-releases depName=etcd-io/etcd digestVersion=v3.5.34
  etcd_3_5_sha256 = "eb10a9f3c8d547c4a76c8c39ab6bcd56e95c8da6aec86d4136e433ef6d39977b"

  etcd_known_versions = {
    "3.7" = { version = local.etcd_3_7_version, sha256 = local.etcd_3_7_sha256 }
    "3.6" = { version = local.etcd_3_6_version, sha256 = local.etcd_3_6_sha256 }
    "3.5" = { version = local.etcd_3_5_version, sha256 = local.etcd_3_5_sha256 }
  }

  is_minor      = can(regex("^[vV]?[0-9]+\\.[0-9]+$", var.etcd_version))
  clean_minor   = trimprefix(var.etcd_version, "v")
  matched_minor = lookup(local.etcd_known_versions, local.clean_minor, null)

  resolved_version = local.is_minor ? try(local.matched_minor.version, null) : var.etcd_version
  resolved_shasum  = coalesce(var.etcd_shasum, try(local.matched_minor.sha256, null), "")
}

module "server_nodes" {
  count                 = var.server_count
  source                = "../node"
  project_name          = var.project_name
  name                  = "${var.name}-node-${count.index}"
  ssh_private_key_path  = var.ssh_private_key_path
  ssh_user              = var.ssh_user
  ssh_tunnels           = count.index == 0 ? var.additional_ssh_tunnels : []
  node_module           = var.node_module
  node_module_variables = var.node_module_variables
  network_config        = var.network_config
}

resource "ssh_sensitive_resource" "node_installation" {
  count               = var.server_count
  host                = module.server_nodes[count.index].private_name
  private_key         = file(var.ssh_private_key_path)
  user                = var.ssh_user
  bastion_host        = var.network_config.ssh_bastion_host
  bastion_user        = var.network_config.ssh_bastion_user
  bastion_private_key = local.bastion_private_key
  timeout             = "600s"

  lifecycle {
    precondition {
      condition     = local.resolved_shasum != ""
      error_message = "A valid SHA256 checksum is required for etcd version '${var.etcd_version}'. Either use a supported minor version (${join(", ", keys(local.etcd_known_versions))}) or provide etcd_shasum."
    }
  }

  file {
    content = templatefile("${path.module}/install_etcd.sh", {
      etcd_version = local.resolved_version,
      etcd_shasum  = local.resolved_shasum,
      etcd_token   = "${var.project_name}-${var.name}-token"
      etcd_name    = "${var.project_name}-${var.name}-${count.index}"
      server_name  = module.server_nodes[count.index].private_name
      server_ip    = module.server_nodes[count.index].private_ip
      etcd_names   = formatlist("%s-%s", "${var.project_name}-${var.name}", range(0, var.server_count))
      server_ips   = [for node in module.server_nodes : node.private_ip]
    })
    destination = "/root/install_etcd.sh"
    permissions = "0700"
  }

  commands = [
    "/root/install_etcd.sh > >(tee install_etcd.log) 2> >(tee install_etcd.err >&2)",
  ]
}
