output "clusters" {
  value = merge({
    "upstream" : module.upstream_cluster.config,
    "tester" : var.tester_cluster != null ? module.tester_cluster[0].config : null,
    },
    { for i, cluster in local.downstream_clusters : cluster.name => module.downstream_clusters[i].config },
  )
}

output "custom_clusters" {
  value = flatten([
    for template_idx, template in var.downstream_cluster_templates : [
      for cluster_idx in range(template.cluster_count): merge(template, {
          nodes = [
            for i, node in local.nodes : module.nodes[i] if node.origin_index == template_idx
          ]
          name = "${local.custom_cluster_name_prefix}-${template_idx}-${cluster_idx}"
          machine_pools = template.machine_pools
          distro_version = template.distro_version
          cluster_count = template.cluster_count
        })
      ] if template.cluster_count > 0 && template.is_custom_cluster
  ])
}
