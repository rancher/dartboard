resource "harvester_image" "opensuse156" {
  count = var.create ? 1 : 0
  name = "${var.project_name}-opensuse156"
  namespace = var.namespace
  display_name = "openSUSE 15.6"
  source_type = "download"
  url = "https://download.opensuse.org/repositories/Cloud:/Images:/Leap_15.6/images/openSUSE-Leap-15.6.x86_64-NoCloud.qcow2"
}
