output "data" {
  value = "#cloud-config\n${yamlencode(local.user_data)}"
}