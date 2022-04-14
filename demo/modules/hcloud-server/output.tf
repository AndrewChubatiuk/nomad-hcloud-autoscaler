output "ipv4_addresses" {
  value = hcloud_server.server.*.ipv4_address
}

output "ids" {
  value = hcloud_server.server.*.id
}