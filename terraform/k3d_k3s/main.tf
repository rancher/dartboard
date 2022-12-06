terraform {
  required_providers {
    k3d = {
      source  = "pvotal-tech/k3d"
      version = "0.0.6"
    }
  }
}

resource "k3d_cluster" "cluster" {
  name    = "${var.project_name}-${var.name}"
  servers = var.server_count
  agents  = var.agent_count

  image   = "docker.io/rancher/k3s:${replace(var.distro_version, "+", "-")}"
  network = var.network_name

  k3d {
    disable_load_balancer = true
  }

  kubeconfig {
    update_default_kubeconfig = true
    switch_current_context    = true
  }

  k3s {
    dynamic "extra_args" {
      for_each = concat([{
        // https://github.com/kubernetes/kubernetes/issues/104459
        arg          = "--disable=metrics-server",
        node_filters = ["all:*"]
        }], [
        for san in var.sans :
        {
          arg          = "--tls-san=${san}",
          node_filters = ["server:*"]
        }
      ])
      content {
        arg          = extra_args.value["arg"]
        node_filters = extra_args.value["node_filters"]
      }
    }
  }

  dynamic "port" {
    for_each = var.additional_port_mappings
    content {
      host_port      = port.value[0]
      container_port = port.value[1]
      node_filters = [
        "server:0:direct",
      ]
    }
  }
}

output "first_server_private_name" {
  value = "k3d-${var.project_name}-${var.name}-server-0"
}
