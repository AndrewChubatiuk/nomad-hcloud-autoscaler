resource "nomad_job" "service" {
  jobspec = file(var.path)
}