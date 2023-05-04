data "hcloud_datacenters" "dc" {}

resource "random_shuffle" "dc" {
  count        = var.server_count
  input        = local.locations
  result_count = 1
}

resource "tls_private_key" "server" {
  algorithm   = "ECDSA"
  ecdsa_curve = "P384"
}

resource "hcloud_ssh_key" "server" {
  name       = "nomad"
  public_key = tls_private_key.server.public_key_openssh
}

resource "hcloud_server" "server" {
  count       = var.server_count
  name        = "${var.prefix}-${count.index}"
  image       = var.image
  datacenter  = join("", random_shuffle.dc[count.index].result)
  ssh_keys    = [hcloud_ssh_key.server.id]
  server_type = var.server_type
  user_data   = var.user_data
  labels      = merge(var.labels, { "Name" = var.prefix })
}

resource "hcloud_server_network" "server" {
  count      = var.server_count
  server_id  = hcloud_server.server[count.index].id
  network_id = var.network_id
}
