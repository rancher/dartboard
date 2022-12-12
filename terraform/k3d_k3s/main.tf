terraform {
  required_providers {
    k3d = {
      source = "pvotal-tech/k3d"
    }
    docker = {
      source = "kreuzwerker/docker"
    }
  }
}

// datastore in docker

resource "docker_image" "mariadb" {
  count = var.datastore == "mariadb" ? 1 : 0
  name  = "mariadb:10.10.2-jammy"
}

// connect with
// mariadb -h 127.0.0.1 -P 3306 -u kineuser --password=kinepassword kine
resource "docker_container" "mariadb" {
  count = var.datastore == "mariadb" ? 1 : 0
  image = docker_image.mariadb[0].image_id
  name  = "kine-mariadb"
  env = [
    "MARIADB_DATABASE=${var.datastore_dbname}",
    "MARIADB_USER=${var.datastore_username}",
    "MARIADB_PASSWORD=${var.datastore_password}",
    "MARIADB_ROOT_PASSWORD=${var.datastore_password}",
  ]
  networks_advanced {
    name = var.network_name
  }

  ports {
    internal = 3306
    external = 3306
  }
}

resource "docker_image" "postgres" {
  count = var.datastore == "postgres" ? 1 : 0
  name  = "postgres:15.1-alpine"
}

// connect with
// PGPASSWORD=kinepassword psql -U kineuser -h localhost kine
resource "docker_container" "postgres" {
  count = var.datastore == "postgres" ? 1 : 0
  image = docker_image.postgres[0].image_id
  name  = "kine-postgres"
  env = [
    "POSTGRES_DB=${var.datastore_dbname}",
    "POSTGRES_USER=${var.datastore_username}",
    "POSTGRES_PASSWORD=${var.datastore_password}",
  ]

  networks_advanced {
    name = var.network_name
  }

  ports {
    internal = 5432
    external = 5432
  }
}

locals {
  datastore_endpoint = (
    var.datastore == "mariadb" ?
    "mysql://${var.datastore_username}:${var.datastore_password}@tcp(kine-mariadb:3306)/${var.datastore_dbname}" :
    var.datastore == "postgres" ?
    "postgres://${var.datastore_username}:${var.datastore_password}@kine-postgres:5432/${var.datastore_dbname}?sslmode=disable" :
    null
  )
}


resource "k3d_cluster" "cluster" {
  depends_on = [docker_container.mariadb, docker_container.postgres]
  name       = "${var.project_name}-${var.name}"
  servers    = var.server_count
  agents     = var.agent_count

  image   = var.image != null ? var.image : "docker.io/rancher/k3s:${replace(var.distro_version, "+", "-")}"
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
        }],
        var.datastore != null ? [{
          arg          = "--datastore-endpoint=${local.datastore_endpoint}",
          node_filters = ["server:*"]
        }] : [],
        var.enable_pprof ? [{
          arg          = "--enable-pprof",
          node_filters = ["server:*"]
        }] : [],
        [
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
