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
      merge(template, {
        nodes = [
          for i, node in local.nodes : merge(module.nodes[i], {
            ssh_user     = var.ssh_user
            ssh_key_path = abspath(pathexpand(var.ssh_private_key_path))
          }) if node.origin_index == template_idx
        ]
        name = "${local.custom_cluster_name_prefix}-${template_idx}"
        machine_pools = [
          for j, pool in template.machine_pools : pool.machine_pool_config
        ]
      })
    ] if template.cluster_count > 0 && template.is_custom_cluster
  ])
}
