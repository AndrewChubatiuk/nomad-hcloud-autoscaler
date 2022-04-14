FROM hashicorp/nomad-autoscaler:0.3.6
ADD bin/hcloud-server /plugins/
