job "redis" {
  datacenters = ["dc1"]
  type        = "service"

  group "redis" {
    count = 15

    constraint {
      distinct_hosts = false
    }

    constraint {
      attribute = "${node.class}"
      operator  = "="
      value     = "redis"
    }

    restart {
      mode = "delay"
    }

    service {
      name = "redis"
      port = "6379"

      check {
        type     = "script"
        name     = "redis"
        task     = "redis_server"
        command  = "/bin/sh"
        args     = ["-c", "[ \"$(redis-cli ping)\" = 'PONG' ] && exit 0; exit 1"]
        interval = "60s"
        timeout  = "5s"

        check_restart {
          limit           = 3
          grace           = "30s"
          ignore_warnings = false
        }
      }
    }

    task "redis_server" {
      driver = "docker"

      config {
        image = "redis:latest"
        sysctl = {
          "net.core.somaxconn" = "1024"
        }
      }

      resources {
        cpu    = 500
        memory = 512
      }
    }
  }
}