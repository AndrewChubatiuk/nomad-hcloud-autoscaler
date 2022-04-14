locals {
  user_data = {
    apt = {
      sources = {
        "hashicorp-releases.list" = {
          keyid  = "9DC858229FC7DD38854AE2D88D81803C0EBFCD88"
          source = "deb [arch=amd64] https://apt.releases.hashicorp.com $RELEASE main"
        }
        "docker.list" = {
          keyid  = "E8A032E094D8EB4EA189D270DA418C88A3219F7B"
          source = "deb [arch=amd64] https://download.docker.com/linux/ubuntu $RELEASE stable"
        }
      }
    }
    packages = [
      "nomad",
      "consul",
      "docker-ce",
      "docker-ce-cli",
      "containerd.io",
      "jq",
    ]
    runcmd = [
      ["/usr/bin/bootstrap.sh"],
    ]
    write_files = [
      {
        path        = "/etc/nomad.d/nomad.hcl"
        permissions = "0644"
        content     = templatefile("${path.module}/data/nomad.hcl.tmpl", var.input)
        }, {
        path        = "/etc/consul.d/consul.hcl"
        permissions = "0644"
        content     = templatefile("${path.module}/data/consul.hcl.tmpl", var.input)
        }, {
        path        = "/usr/bin/bootstrap.sh"
        permissions = "0755"
        content     = templatefile("${path.module}/data/bootstrap.sh", var.input)
      }
    ]
  }
}

variable "input" {}