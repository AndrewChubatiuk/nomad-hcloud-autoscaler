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
}

module "server_user_data" {
  source = "../modules/user-data"
  input = {
    "node_class"   = "nomad-server"
    "datacenter"   = "dc1"
    "servers"      = []
    "interface"    = "eth0"
    "consul_token" = ""
    "nomad_token"  = ""
  }
}

module "redis_client_user_data" {
  source = "../modules/user-data"
  input = {
    "node_class"   = "redis"
    "datacenter"   = "dc1"
    "servers"      = module.nomad.ipv4_addresses
    "interface"    = "eth0"
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
    host        = module.nomad.ipv4_addresses[0]
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
      "/tmp/wait.sh http://${module.nomad.ipv4_addresses[0]}:4646/v1/status/leader",
    ]
  }

  provisioner "remote-exec" {
    inline = [
      "chmod +x /tmp/wait.sh",
      "/tmp/wait.sh http://${module.nomad.ipv4_addresses[0]}:8500/v1/status/leader",
    ]
  }

  provisioner "local-exec" {
    command = "scp -q -i ${local_sensitive_file.ssh.filename} root@${module.nomad.ipv4_addresses[0]}:/tmp/creds.json ${path.root}/creds.json"
  }
}

data "local_sensitive_file" "creds" {
  filename   = "${path.root}/creds.json"
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
