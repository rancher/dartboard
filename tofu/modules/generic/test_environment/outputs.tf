output "clusters" {
  value = merge({
    "upstream" : module.upstream_cluster.config,
    "tester" : var.deploy_tester_cluster ? module.tester_cluster[0].config : null,
    },
    { for i, cluster in local.downstream_clusters : cluster.name => module.downstream_clusters[i].config }
  )
}
