locals {
  downstream_clusters = flatten([
    for i, template in var.downstream_cluster_templates : [
      for j in range(template.cluster_count) : merge(template, { name = "${template.name_prefix}${i}-${j}" })
  ] if template.cluster_count > 0 ])
  all_clusters = flatten(concat([var.upstream_cluster],
    local.downstream_clusters,
    var.deploy_tester_cluster ? [var.tester_cluster] : []
  ))


  k3s_clusters  = [for i, cluster in local.all_clusters : merge(cluster, {name = "${cluster.name_prefix}-${i}"}) if strcontains(cluster.distro_version, "k3s")]
  rke2_clusters = [for i, cluster in local.all_clusters : merge(cluster, {name = "${cluster.name_prefix}-${i}"}) if strcontains(cluster.distro_version, "rke2")]
}
