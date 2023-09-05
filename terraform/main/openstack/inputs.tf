locals {
  project_name = "st"

  upstream_cluster = {
    name           = "upstream"
    server_count   = 1
    agent_count    = 2
    distro_version = "v1.24.12+k3s1"
    agent_labels = [
      [{ key : "monitoring", value : "true" }]
    ]
    agent_taints = [
      [{ key : "monitoring", value : "true", effect : "NoSchedule" }]
    ]

    // openstack-specific
    flavor_name = "c2-15"                                // OVHcloud: 4vCPUs at 3 GHz, 15 GiB RAM
    image_id    = "4e425abb-c9b0-4912-8f52-8ad114031b5c" // OVHcloud: GRA7 - CentOS7
  }

  downstream_clusters = [
    for i in range(10) :
    {
      name           = "downstream-${i}"
      server_count   = 1
      agent_count    = 10
      distro_version = "v1.24.12+k3s1"
      agent_labels   = []
      agent_taints   = []

      // openstack-specific
      flavor_name = "c2-7"                                 // OVHcloud: 2vCPUs at 3 GHz, 7 GiB RAM
      image_id    = "4e425abb-c9b0-4912-8f52-8ad114031b5c" // OVHcloud: GRA7 - CentOS7
    }
  ]

  tester_cluster = {
    name           = "tester"
    server_count   = 1
    agent_count    = 0
    distro_version = "v1.24.12+k3s1"
    agent_labels   = []
    agent_taints   = []

    // openstack-specific
    flavor_name = "b2-7"                                 // OVHcloud: 2vCPUs at 2 GHz, 7 GiB RAM
    image_id    = "4e425abb-c9b0-4912-8f52-8ad114031b5c" // OVHcloud: GRA7 - CentOS7
  }

  clusters = concat([local.upstream_cluster], local.downstream_clusters, [local.tester_cluster])

  // openstack-specific
  availability_zone           = "nova"
  ip_wildcard_resolver_domain = "nip.io" // Any NIP.io like "DNS Crafter" - 1.2.3.4.nip.io resolves to 1.2.3.4

  // networking
  // by default a new private network will be created, uncomment below to attach to an existing private network
  network_id = null
  // network_id = "e3a21534-b3a5-44c0-8bb4-de72061471d9"

  # subnet definition
  subnet_cidr     = "10.3.0.0/16"
  dns_nameservers = ["213.186.33.99"] // OVHcloud DNS server

  bastion_flavor = "d2-4"                                 // OVHcloud: 1vCPU, 4 GiB RAM
  bastion_image  = "4e425abb-c9b0-4912-8f52-8ad114031b5c" // OVHcloud: GRA7 - CentOS7

  // External Network - public network ID
  external_network_id = "393d06cc-a82c-4dc4-a576-c79e8dd67ba3" // OVHcloud: Ext-Net - GRA7
  // External Network - public network **Name**
  floating_ip_pool_ext = "Ext-Net"
}

variable "ssh_public_key_path" {
  description = "Path to SSH public key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519.pub"
}

variable "ssh_private_key_path" {
  description = "Path to SSH private key file (can be generated with `ssh-keygen -t ed25519`)"
  default     = "~/.ssh/id_ed25519"
}
