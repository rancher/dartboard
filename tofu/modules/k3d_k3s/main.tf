terraform {
  required_providers {
    k3d = {
      source = "moio/k3d"
    }
    docker = {
      source = "kreuzwerker/docker"
    }
  }
}

// datastore in docker

resource "docker_image" "mariadb" {
  count        = var.datastore == "mariadb" ? 1 : 0
  name         = "mariadb:10.10.2-jammy"
  keep_locally = true
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

  volumes {
    container_path = "/var/lib/mysql"
    host_path      = "/tmp/${var.project_name}-kine-data/mysql"
  }
  remove_volumes = false
}

resource "docker_image" "postgres" {
  count        = var.datastore == "postgres" ? 1 : 0
  name         = "postgres:15.1-alpine"
  keep_locally = true
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
    "POSTGRES_INITDB_ARGS=--locale=C",
  ]

  networks_advanced {
    name = var.network_name
  }

  ports {
    internal = 5432
    external = 5432
  }

  volumes {
    container_path = "/var/lib/postgresql/data"
    host_path      = "/tmp/${var.project_name}-kine-data/postgres"
  }
  remove_volumes = false

  healthcheck {
    test = [
      "CMD-SHELL",
      "PGPASSWORD=${var.datastore_password} pg_isready --dbname=${var.datastore_dbname} --username=${var.datastore_username}"
    ]
    interval = "1s"
    retries  = "60"
    timeout  = "10s"
  }
  wait = true

  command = [
    "postgres",
    "-c",
    "log_min_duration_statement=${var.postgres_log_min_duration_statement != null ? var.postgres_log_min_duration_statement : -1}",

    // rough minimal tuning parameters below generated via https://pgtune.leopard.in.ua/
    // assumptions: web application, 16 GB RAM, 4 vCPUs, SSD storage, 50 connections

    "-c", "max_connections=50",
    "-c", "shared_buffers=4GB",
    "-c", "effective_cache_size=12GB",
    "-c", "maintenance_work_mem=1GB",
    "-c", "checkpoint_completion_target=0.9",
    "-c", "wal_buffers=16MB",
    "-c", "default_statistics_target=100",
    "-c", "random_page_cost=1.1",
    "-c", "effective_io_concurrency=200",
    "-c", "work_mem=41943kB",
    "-c", "min_wal_size=1GB",
    "-c", "max_wal_size=4GB",
    "-c", "max_worker_processes=4",
    "-c", "max_parallel_workers_per_gather=2",
    "-c", "max_parallel_workers=4",
    "-c", "max_parallel_maintenance_workers=2",
  ]

  shm_size = 5 * 1024 // MiB, has to be more than shared_buffers above
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

resource "docker_container" "kine" {
  depends_on = [docker_container.mariadb, docker_container.postgres]
  count      = var.datastore == "mariadb" || var.datastore == "postgres" ? 1 : 0
  image      = var.kine_image
  name       = "kine"

  networks_advanced {
    name = var.network_name
  }

  ports {
    internal = 2379
    external = 2379
  }

  command = concat([
    "--endpoint",
    local.datastore_endpoint,
    ],
  var.kine_debug ? ["--debug"] : [])
}

resource "k3d_cluster" "cluster" {
  count      = var.server_count > 0 ? 1 : 0
  depends_on = [docker_container.kine]
  name       = "${var.project_name}-${var.name}"
  servers    = var.server_count
  agents     = var.agent_count

  // hardcode, so that cluster can be re-created and run from previous datastore
  token = "secretToken"

  image   = var.image != null ? var.image : "docker.io/rancher/k3s:${replace(var.distro_version, "+", "-")}"
  network = var.network_name

  k3d {
    disable_load_balancer = true
  }

  kubeconfig {
    update_default_kubeconfig = false
    switch_current_context    = false
  }

  volume {
    source       = "/sys"
    destination  = "/host/sys"
    node_filters = []
  }

  volume {
    source       = "/var/log/k3d/audit"
    destination  = "/var/log/kubernetes/audit"
    node_filters = ["server:*"]
  }

  volume {
    source       = "/var/lib/k3d/audit"
    destination  = "/var/lib/rancher/k3s/server/manifests/audit"
    node_filters = ["server:*"]
  }

  k3s {
    dynamic "extra_args" {
      for_each = concat(
        var.enable_metrics == false ? [
          {
            arg          = "--disable=metrics-server",
            node_filters = ["server:*"]
          }
        ] : [],
        // if datastore requires an external kine instance, point to it
        var.datastore == "mariadb" || var.datastore == "postgres" ? [
          {
            arg          = "--datastore-endpoint=http://kine:2379",
            node_filters = ["server:*"]
          }
        ] : [],
        // normally k3s defaults to sqlite for 1-node clusters and embedded etcd for multi-node ones
        // it is possible to force use of the embedded etcd for 1-node clusters via --cluster-init
        var.datastore == "embedded_etcd" && var.server_count == 1 && var.agent_count == 0 ? [
          {
            arg          = "--cluster-init",
            node_filters = ["server:0"]
          }
        ] : [],
        var.enable_pprof ? [
          {
            arg          = "--enable-pprof",
            node_filters = ["server:*"]
          }
        ] : [],
        var.log_level != null ? [
          {
            arg          = "-v=${var.log_level}",
            node_filters = ["server:*"]
          }
        ] : [],
        [
          for san in var.sans :
          {
            arg          = "--tls-san=${san}",
            node_filters = ["server:*"]
          }
        ],
        var.enable_audit_log ? flatten([
          {
            arg          = "--kube-apiserver-arg=audit-policy-file=/var/lib/rancher/k3s/server/manifests/audit/audit.yaml",
            node_filters = ["server:*"]
          },
          [for i in range(0, var.server_count) : {
            arg          = "--kube-apiserver-arg=audit-log-path=/var/log/kubernetes/audit/audit_server_${i}.log",
            node_filters = ["server:${i}"]
          }],
        ]) : [],
        flatten([
          for agent_i, labels in var.agent_labels :
          [
            for label in labels :
            {
              arg          = "--node-label=${label.key}=${label.value}",
              node_filters = ["agent:${agent_i}"]
            }
          ]
        ]),
        flatten([
          for agent_i, taints in var.agent_taints :
          [
            for taint in taints :
            {
              arg          = "--node-taint=${taint.key}=${taint.value}:${taint.effect}",
              node_filters = ["agent:${agent_i}"]
            }
          ]
        ]),
      )
      content {
        arg          = extra_args.value["arg"]
        node_filters = extra_args.value["node_filters"]
      }
    }
  }

  registries {
    config = yamlencode({
      mirrors = {
        for registry in var.pull_proxy_registries :
        registry.name => { endpoints = ["http://${registry.address}"] }
      }
    })
  }

  env {
    key          = "GOGC"
    value        = tostring(var.gogc)
    node_filters = ["server:*"]
  }

  kube_api {
    host_port = var.kubernetes_api_port
  }

  dynamic "port" {
    for_each = concat([
      {
        host_port : var.app_http_port,
        container_port : 80,
        node_filters : ["server:0:direct"]
      },
      {
        host_port : var.app_https_port,
        container_port : 443,
        node_filters : ["server:0:direct"]
      },
      ],
      var.enable_metrics ?
      [
        for i in range(0, var.server_count) :
        {
          host_port : var.first_metrics_port + i,
          container_port : 10250,
          node_filters = ["server:${i}:direct"]
        }
      ] : [],
      var.enable_metrics ?
      [
        for i in range(0, var.agent_count) :
        [
          {
            host_port : var.first_metrics_port + var.server_count + i,
            container_port : 10250,
            node_filters = ["agent:${i}:direct"]
          }
        ]
      ] : [],
    )
    content {
      host_port      = port.value["host_port"]
      container_port = port.value["container_port"]
      node_filters   = port.value["node_filters"]
    }
  }
}
