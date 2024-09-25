resource "harvester_ssh_key" "public_key" {
  name      = "${var.project_name}-key"
  namespace = var.namespace

  public_key = file(var.ssh_public_key_path)
}
