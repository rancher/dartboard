locals {
  all_clusters = concat([{
    name                        = "upstream"
    server_count                = var.upstream_server_count
    agent_count                 = var.upstream_agent_count
    distro_version              = var.upstream_distro_version
    reserve_node_for_monitoring = var.upstream_reserve_node_for_monitoring
    public_ip     = var.upstream_public_ip
    instance_type = var.upstream_instance_type
    ami           = var.upstream_ami
  }],
    [for i in range(var.downstream_cluster_count) :
      {
        name                        = "downstream-${i}"
        server_count                = var.downstream_server_count
        agent_count                 = var.downstream_agent_count
        distro_version              = var.downstream_distro_version
        reserve_node_for_monitoring = false
        public_ip     = var.downstream_public_ip
        instance_type = var.downstream_instance_type
        ami           = var.downstream_ami
      }],
      var.deploy_tester_cluster ? [{
      name                        = "tester"
      server_count                = var.tester_server_count
      agent_count                 = var.tester_agent_count
      distro_version              = var.tester_distro_version
      reserve_node_for_monitoring = false
      public_ip     = var.tester_public_ip
      instance_type = var.tester_instance_type
      ami           = var.tester_ami
    }] : []
  )

  k3s_clusters  = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "k3s")]
  rke_clusters  = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "rke_")]
  rke2_clusters = [for cluster in local.all_clusters : cluster if strcontains(cluster.distro_version, "rke2")]
}
