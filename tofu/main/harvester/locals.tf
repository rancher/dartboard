locals {
  downstream_clusters = [
    for i, template in var.downstream_cluster_templates : [
      for j in range(template.cluster_count) : merge(template, { name_prefix = "${template.name_prefix}${i}-${j}" })
  ]]
  all_clusters = flatten(concat([var.upstream_cluster],
    local.downstream_clusters,
    var.deploy_tester_cluster ? [var.tester_cluster] : []
  ))

  k3s_clusters  = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "k3s")]
  rke2_clusters = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "rke2")]
}
