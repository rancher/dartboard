locals {
  downstream_clusters = flatten([
    for i, template in var.downstream_cluster_templates : [
      for j in range(template.cluster_count) : merge(template, { name_prefix = "${template.name_prefix}${i}-${j}" })
  ] if template.cluster_count > 0 ])
  all_clusters = flatten(concat([var.upstream_cluster],
    local.downstream_clusters,
    var.deploy_tester_cluster ? [var.tester_cluster] : []
  ))

  k3s_clusters  = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "k3s")]
  rke2_clusters = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "rke2")]
  create_image = ((length(local.k3s_clusters) > 0 && anytrue([for c in local.k3s_clusters : c.image_name == null])) ||
   (length(local.rke2_clusters) > 0 && anytrue([for c in local.rke2_clusters : c.image_name == null])))
}
