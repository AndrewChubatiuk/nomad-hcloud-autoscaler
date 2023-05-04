provider "hcloud" {
  token = var.hcloud_token
}

provider "nomad" {
  address      = "http://${data.ssh_tunnel.nomad.local.address}"
  secret_id    = jsondecode(data.local_sensitive_file.creds.content)["nomad"]
  consul_token = jsondecode(data.local_sensitive_file.creds.content)["consul"]

}

provider "consul" {
  address = "http://${data.ssh_tunnel.consul.local.address}"
  token   = jsondecode(data.local_sensitive_file.creds.content)["consul"]
}

provider "ssh" {
  user = "root"
  auth = {
    private_key = {
      content = module.nomad.private_key
    }
  }
  server = {
    host = module.nomad.public_ips[0]
    port = 22
  }
}
