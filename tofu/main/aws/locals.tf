locals {
  downstream_clusters = [
    for i, config in var.downstream_clusters : [
      for c in range(config.quantity) : merge(config,{name_prefix = "${config.name_prefix}${i}-${c}"})
    ]]
  all_clusters = flatten(concat([var.upstream_cluster],
    local.downstream_clusters,
    var.deploy_tester_cluster ? [var.tester_cluster] : []
  ))

  k3s_clusters  = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "k3s")]
  rke2_clusters = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "rke2")]
}
