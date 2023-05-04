locals {
  user_data = {
    apt = {
      sources = {
        "hashicorp-releases.list" = {
          keyid  = "798A EC65 4E5C 1542 8C8E 42EE AA16 FCBC A621 E701"
          source = "deb [signed-by=$KEY_FILE] https://apt.releases.hashicorp.com $RELEASE main"
        }
        "docker.list" = {
          keyid  = "9DC8 5822 9FC7 DD38 854A E2D8 8D81 803C 0EBF CD88"
          source = "deb [signed-by=$KEY_FILE] https://download.docker.com/linux/ubuntu $RELEASE stable"
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
