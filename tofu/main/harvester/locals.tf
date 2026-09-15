locals {
  node_module_variables = setunion(
    var.downstream_cluster_templates[*].node_module_variables,
    var.tester_cluster != null ? [var.tester_cluster.node_module_variables] : [],
    var.upstream_cluster != null ? [var.upstream_cluster.node_module_variables] : []
  )

  imageNames = toset([
    for i, node_module_variables in local.node_module_variables : join("/", [node_module_variables.image_namespace != null ? node_module_variables.image_namespace : var.namespace], [node_module_variables.image_name])
    if lookup(node_module_variables, "image_name", null) != null
  ])

  sshKeyNames = toset(flatten([
    for i, node_module_variables in local.node_module_variables : [
      for i, key in node_module_variables.ssh_shared_public_keys : join("/", [key.namespace, key.name])
    ] if lookup(node_module_variables, "ssh_shared_public_keys", null) != null
  ]))

  images_by_name = {
    for image, data_obj in data.harvester_image.images_by_name :
    image => data_obj.id
  }

  ssh_keys_by_name = {
    for key, data_obj in data.harvester_ssh_key.ssh_keys :
    key => tomap({
      id         = data_obj.id,
      public_key = data_obj.public_key
    })
  }
}

data "harvester_image" "images_by_name" {
  for_each = length(local.imageNames) > 0 ? {
    for i, image in local.imageNames :
    image => {
      namespace    = split("/", image)[0],
      display_name = split("/", image)[1],
    }
  } : {}
  display_name = each.value.display_name
  namespace    = each.value.namespace
}

data "harvester_ssh_key" "ssh_keys" {
  for_each = length(local.sshKeyNames) > 0 ? {
    for i, key in local.sshKeyNames :
    key => {
      namespace = split("/", key)[0],
      name      = split("/", key)[1],
    }
  } : {}
  name      = each.value.name
  namespace = each.value.namespace
}