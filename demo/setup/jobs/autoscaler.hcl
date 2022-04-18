job "autoscaler" {
  datacenters = ["dc1"]

  group "autoscaler" {
    count = 1

    constraint {
      attribute = "${node.class}"
      operator  = "="
      value     = "nomad-server"
    }

    task "autoscaler" {
      driver = "docker"

      config {
        image      = "achubatiuk/nomad-autoscaler:fix-scaler"
        command    = "nomad-autoscaler"
        args       = ["agent", "-config", "${NOMAD_TASK_DIR}/config.hcl", "-plugin-dir", "/plugins"]
        force_pull = true
      }

      template {
        data          = <<EOF
scaling  "cluster_class-batch" {
  enabled = true
  min     = 1
  max     = 5

  policy {
    cooldown            = "1m"
    evaluation_interval = "1m"

    check "scaling" {
      source = "prometheus"
      query  = "ceil(nomad_nomad_job_summary_queued{task_group=~\"redis\"}/2)"

      strategy "pass-through" {}
    }

    target "hcloud-server" {
      node_class                   = "redis"
      datacenter                   = "dc1"
      hcloud_ssh_keys              = "nomad"
      hcloud_group_id              = "redis"
      hcloud_labels                = "key1=value1"
      hcloud_location              = "fsn1"
      hcloud_user_data             = "{{ key "secrets/nomad/redis/user-data" }}"
      hcloud_b64_user_data_encoded = "true"
    }
  }
}
           EOF
        destination   = "${NOMAD_TASK_DIR}/policies/hcloud.hcl"
        change_mode   = "signal"
        change_signal = "SIGHUP"
      }

      template {
        data = <<EOF
enable_debug = true
log_level = "debug"
nomad {
  address = "http://{{ env "attr.unique.network.ip-address" }}:4646"
  token   = "{{ key "secrets/nomad/token" }}"
}
apm "nomad" {
  driver = "nomad-apm"
  config = {
    address = "http://{{ env "attr.unique.network.ip-address" }}:4646"
    token   = "{{ key "secrets/nomad/token" }}"
  }
}
{{ range service "prometheus" }}         
apm "prometheus" {
  driver = "prometheus"
  config = {
    address = "http://{{ .Address }}:{{ .Port }}"
  }
}
{{ end }}
strategy "pass-through" {
  driver = "pass-through"
}

target "hcloud-server" {
  driver = "hcloud-server"
  config = {
    hcloud_token = "{{ key "secrets/hcloud/token" }}"
  }
}
policy {
  dir              = "{{ env "NOMAD_TASK_DIR" }}/policies"
  default_cooldown = "1m"
}
          EOF

        destination = "${NOMAD_TASK_DIR}/config.hcl"
      }
    }
  }
}
