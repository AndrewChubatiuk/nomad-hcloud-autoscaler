variable "ssh_key" {
  default = "~/.ssh/id_rsa"
}

variable "hcloud_token" {
  sensitive = true
}