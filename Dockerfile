FROM hashicorp/nomad-autoscaler:0.3.7
ADD bin/hcloud-server /plugins/
