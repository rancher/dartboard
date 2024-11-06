terraform {
  required_version = "1.8.2"
  required_providers {
    openstack = {
      source  = "terraform-provider-openstack/openstack"
      version = "1.52.1"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "4.0.3"
    }
    helm = {
      source  = "hashicorp/helm"
      version = "2.7.1"
    }
    ssh = {
      source  = "loafoe/ssh"
      version = "2.2.1"
    }
  }
}

resource "openstack_compute_keypair_v2" "main_keypair" {
  name       = "${local.project_name}-main-keypair"
  public_key = file(var.ssh_public_key_path)
}

module "network" {
  source               = "../../modules/openstack_network"
  bastion_flavor       = local.bastion_flavor
  bastion_image        = local.bastion_image
  project_name         = local.project_name
  network_id           = local.network_id
  subnet_cidr          = local.subnet_cidr
  availability_zone    = local.availability_zone
  keypair              = openstack_compute_keypair_v2.main_keypair.name
  ssh_public_key_path  = var.ssh_public_key_path
  ssh_private_key_path = var.ssh_private_key_path
  dns_nameservers      = local.dns_nameservers
  floating_ip_pool_ext = local.floating_ip_pool_ext
  external_network_id  = local.external_network_id
}

module "cluster" {
  depends_on   = [module.network]
  count        = length(local.clusters)
  source       = "../../modules/openstack_k3s"
  project_name = local.project_name
  name         = local.clusters[count.index].name
  server_count = local.clusters[count.index].server_count
  agent_count  = local.clusters[count.index].agent_count
  agent_labels = local.clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true" }]
  ] : []
  agent_taints = local.clusters[count.index].reserve_node_for_monitoring ? [
    [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
  ] : []
  distro_version = local.clusters[count.index].distro_version

  image_id             = local.clusters[count.index].image_id
  flavor_name          = local.clusters[count.index].flavor_name
  floating_ip_pool_ext = local.floating_ip_pool_ext
  availability_zone    = local.availability_zone
  keypair              = openstack_compute_keypair_v2.main_keypair.name
  ssh_private_key_path = var.ssh_private_key_path
  ssh_bastion_host     = module.network.bastion_public_name
  network_id           = module.network.private_network_id
  subnet_id            = module.network.private_subnet_id
}
