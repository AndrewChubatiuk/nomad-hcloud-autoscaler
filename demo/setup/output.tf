output "credentials" {
  value = {
    nomad = {
      NOMAD_ADDR  = "http://${module.nomad.ipv4_addresses[0]}:4646"
      NOMAD_TOKEN = jsondecode(data.local_file.creds.content)["nomad"]
    }
    consul = {
      CONSUL_HTTP_ADDR  = "http://${module.nomad.ipv4_addresses[0]}:8500"
      CONSUL_HTTP_TOKEN = jsondecode(data.local_file.creds.content)["consul"]
    }
  }
}