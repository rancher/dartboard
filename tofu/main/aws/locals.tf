locals {
  all_clusters = flatten(concat([var.upstream_cluster],
    var.downstream_clusters,
    var.deploy_tester_cluster ? [var.tester_cluster] : []
  ))

  k3s_clusters  = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "k3s")]
  rke2_clusters = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "rke2")]
}
