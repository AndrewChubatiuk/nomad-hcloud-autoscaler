resource "hcloud_network" "network" {
  name     = "nomad"
  ip_range = var.cidr
}

resource "hcloud_network_subnet" "subnet" {
  count        = var.subnets
  network_id   = hcloud_network.network.id
  type         = "cloud"
  network_zone = "eu-central"
  ip_range     = cidrsubnet(hcloud_network.network.ip_range, var.netnum, count.index)
}
