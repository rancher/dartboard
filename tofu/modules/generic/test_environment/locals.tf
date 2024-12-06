locals {
  unnamed_downstream_clusters = flatten([
    for i, template in var.downstream_cluster_templates : [
      for j in range(template.cluster_count) : template
  ] if template.cluster_count > 0 ])

  downstream_clusters = [for i, cluster in local.unnamed_downstream_clusters : merge(cluster, {name = "downstream-${i}"})]

  all_clusters = flatten(concat(
    [merge(var.upstream_cluster, {name = "upstream"})],
    local.downstream_clusters,
    var.deploy_tester_cluster ? [[merge(var.tester_cluster, {name = "tester"})]] : []
  ))

  k3s_clusters  = [for i, cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "k3s")]
  rke2_clusters = [for i, cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "rke2")]
}
