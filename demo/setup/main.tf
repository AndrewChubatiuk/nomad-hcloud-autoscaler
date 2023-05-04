module "network" {
  source = "../modules/hcloud-network"
}

module "nomad" {
  source       = "../modules/hcloud-server"
  prefix       = "nomad-server"
  location     = "fsn1"
  server_count = 1
  user_data    = module.server_user_data.data
  labels = {
    Environment = "demo"
    Role        = "server"
  }
  server_type = "cx11"
  network_id  = module.network.id
}

data "ssh_tunnel" "nomad" {
  remote = {
    host = module.nomad.private_ips[0]
    port = 4646
  }
  local = {
    port = 4646
  }
  depends_on = [null_resource.nomad]
}

data "ssh_tunnel" "consul" {
  remote = {
    host = module.nomad.private_ips[0]
    port = 8500
  }
  local = {
    port = 8500
  }
  depends_on = [null_resource.nomad]
}

module "server_user_data" {
  source = "../modules/user-data"
  input = {
    "node_class"   = "nomad-server"
    "datacenter"   = "dc1"
    "servers"      = []
    "interface"    = "ens10"
    "consul_token" = ""
    "nomad_token"  = ""
  }
}

module "redis_client_user_data" {
  source = "../modules/user-data"
  input = {
    "node_class"   = "redis"
    "datacenter"   = "dc1"
    "servers"      = module.nomad.private_ips
    "interface"    = "ens10"
    "consul_token" = jsondecode(data.local_sensitive_file.creds.content)["consul"]
    "nomad_token"  = jsondecode(data.local_sensitive_file.creds.content)["nomad"]
  }
}

resource "local_sensitive_file" "ssh" {
  content         = module.nomad.private_key
  filename        = "${path.module}/nomad.pem"
  file_permission = "0400"
}

resource "null_resource" "nomad" {

  triggers = {
    servers = join(",", module.nomad.ids)
  }

  connection {
    host        = module.nomad.public_ips[0]
    user        = "root"
    private_key = module.nomad.private_key
  }

  provisioner "file" {
    source      = "${path.module}/wait.sh"
    destination = "/tmp/wait.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/wait.sh",
      "/tmp/wait.sh http://${module.nomad.private_ips[0]}:8500/v1/status/leader",
    ]
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/wait.sh",
      "/tmp/wait.sh http://${module.nomad.private_ips[0]}:4646/v1/status/leader",
    ]
  }

  provisioner "local-exec" {
    command = "scp -o \"StrictHostKeyChecking no\" -i ${local_sensitive_file.ssh.filename} root@${module.nomad.public_ips[0]}:/tmp/api-tokens.json ${path.root}/api-tokens.json"
  }
}

data "local_sensitive_file" "creds" {
  filename   = "${path.root}/api-tokens.json"
  depends_on = [null_resource.nomad]
}

resource "consul_key_prefix" "secrets" {
  path_prefix = "secrets/"

  subkeys = {
    "nomad/redis/user-data" = base64encode(module.redis_client_user_data.data)
    "hcloud/token"          = var.hcloud_token
    "consul/token"          = jsondecode(data.local_sensitive_file.creds.content)["consul"]
    "nomad/token"           = jsondecode(data.local_sensitive_file.creds.content)["nomad"]
  }
}

module "services" {
  source   = "../modules/nomad-service"
  for_each = fileset("${path.module}/jobs", "*.hcl")
  path     = "${path.module}/jobs/${each.key}"
}
