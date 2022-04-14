variable "location" {
  type    = string
  default = null
}

variable "labels" {
  type    = map(string)
  default = {}
}

variable "prefix" {}

variable "server_count" {}

variable "ssh_keys" {
  type = list(string)
}

variable "server_type" {}

variable "user_data" {}

variable "image" {
  default = "ubuntu-20.04"
}

locals {
  locations = [for dc in data.hcloud_datacenters.dc.names : dc if var.location != null && can(regex("^${var.location}-dc\\d+$", dc))]
}
