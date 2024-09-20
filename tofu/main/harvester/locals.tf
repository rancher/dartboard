locals {
  downstream_clusters = [
    for i, template in var.downstream_cluster_templates : [
      for j in range(template.cluster_count) : merge(template,{name_prefix = "${template.name_prefix}${i}-${j}"})
    ]]
  all_clusters = flatten(concat([var.upstream_cluster],
    local.downstream_clusters,
    var.deploy_tester_cluster ? [var.tester_cluster] : []
  ))

  k3s_clusters  = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "k3s")]
  rke2_clusters = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "rke2")]

  cloudinit_secrets = [for secret in var.cloudinit_secrets: {name = secret.name, namespace = secret.namespace, user_data = file(secret.user_data)}]

  # first_server_public_names = [ for cluster in module.rke2_cluster: cluster.first_server_public_network_interfaces[0].ip_address]
  # first_server_private_names =  [ for cluster in module.rke2_clusters: cluster.first_server_private_network_interfaces[0].ip_address]
}
