# nomad-hcloud-autoscaler

## Demo
Run `terraform apply` in [demo](demo/setup) folder to create: 
 - nomad server which runs services for:
    - nomad-autoscaler
    - prometheus
    - redis

Autoscaler scales hcloud nodes for redis. After successful run both Nomad and Consul are wide-world open and credentials for both you can find in terraform output


## Configuration

`config.hcl`
```
template {
    data = <<-EOF
    nomad {
        address = "http://{{env "attr.unique.network.ip-address" }}:4646"
    }

    telemetry {
        prometheus_metrics = true
        disable_hostname   = true
    }

    apm "prometheus" {
        driver = "prometheus"
        config = {
            address = "http://{{ range service "prometheus" }}{{ .Address }}:{{ .Port }}{{ end }}"
        }
    }

    strategy "target-value" {
        driver = "target-value"
        
    }

    target "hcloud-server" {
        driver = "hcloud-server"
        config = {
            hcloud_token = "YOUR_HCLOUD_TOKEN"
        }
    }
    
    EOF

    destination = "${NOMAD_TASK_DIR}/config.hcl"
    change_mode = "signal"
    change_signal = "SIGHUP"
}
```

`policy`

```
template {
    data = <<-EOF
    scaling  "cluster_class-batch" {
        enabled = true
        min     = 1
        max     = 2

        policy {
        cooldown            = "5m"
        evaluation_interval = "5m"

        check "test-scale" {
            source = "prometheus"
            query  = "YOUR_DESIRED_METRIC"

            strategy "target-value" {
            target = 2
            }
        }

        target "hcloud-server" {
            // datacenter = "XXX"
            node_class = "XXX"
            dry-run             = "false"
            // node_selector_strategy = "newest_create_index"
            hcloud_location = "XXX"
            hcloud_image = "XXX"
            hcloud_user_data = ""
            hcloud_ssh_keys = "XXX"
            hcloud_server_type = "cx11"
            hcloud_group_id = "XXX"
            hcloud_labels = "XXX_node=true"
            hcloud_networks = "XXX"
        }
        }
    }
    EOF
    destination = "${NOMAD_TASK_DIR}/policies/hcloud.hcl"
    change_mode = "signal"
    change_signal = "SIGHUP"
}
```
