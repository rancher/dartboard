locals {
  all_clusters = concat([{
    name                        = "upstream"
    server_count                = var.upstream_server_count
    agent_count                 = var.upstream_agent_count
    distro_version              = var.distro_version
    reserve_node_for_monitoring = var.upstream_reserve_node_for_monitoring
    enable_metrics              = var.upstream_enable_metrics
    }],
    [for i in range(var.downstream_cluster_count) :
      {
        name                        = "downstream-${i}"
        server_count                = var.downstream_server_count
        agent_count                 = var.downstream_agent_count
        distro_version              = var.distro_version
        reserve_node_for_monitoring = false
        enable_metrics              = false
    }],
    var.deploy_tester_cluster ? [{
      name                        = "tester"
      server_count                = var.tester_server_count
      agent_count                 = var.tester_agent_count
      distro_version              = var.distro_version
      reserve_node_for_monitoring = false
      enable_metrics              = false
    }] : []
  )
}
