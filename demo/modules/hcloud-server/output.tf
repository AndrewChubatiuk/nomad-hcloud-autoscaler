output "private_ips" {
  value = hcloud_server_network.server.*.ip
}

output "public_ips" {
  value = hcloud_server.server.*.ipv4_address
}

output "ids" {
  value = hcloud_server.server.*.id
}

output "private_key" {
  value = tls_private_key.server.private_key_pem
}
