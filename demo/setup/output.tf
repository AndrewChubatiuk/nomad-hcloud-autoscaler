output "credentials" {
  sensitive = true
  value = {
    nomad = {
      NOMAD_ADDR  = "http://${module.nomad.private_ips[0]}:4646"
      NOMAD_TOKEN = jsondecode(data.local_sensitive_file.creds.content)["nomad"]
    }
    consul = {
      CONSUL_HTTP_ADDR  = "http://${module.nomad.private_ips[0]}:8500"
      CONSUL_HTTP_TOKEN = jsondecode(data.local_sensitive_file.creds.content)["consul"]
    }
  }
}
