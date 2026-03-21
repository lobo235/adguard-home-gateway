job "adguard-home-gateway" {
  node_pool   = "default"
  datacenters = ["dc1"]
  type        = "service"

  group "adguard-home-gateway" {
    count = 1

    network {
      port "http" {
        to = 8080
      }
    }

    service {
      name     = "adguard-home-gateway"
      port     = "http"
      provider = "consul"
      tags = [
        "traefik.enable=true",
        "traefik.http.routers.adguard-home-gateway.rule=Host(`adguard-home-gateway.example.com`)",
        "traefik.http.routers.adguard-home-gateway.entrypoints=websecure",
        "traefik.http.routers.adguard-home-gateway.tls=true",
      ]

      check {
        type     = "http"
        path     = "/health"
        port     = "http"
        interval = "30s"
        timeout  = "5s"

        check_restart {
          limit = 3
          grace = "30s"
        }
      }
    }

    restart {
      attempts = 3
      interval = "2m"
      delay    = "15s"
      mode     = "fail"
    }

    vault {
      cluster     = "default"
      change_mode = "noop"
    }

    task "adguard-home-gateway" {
      driver = "docker"

      config {
        image = "ghcr.io/lobo235/adguard-home-gateway:latest"
        ports = ["http"]
      }

      template {
        data = <<EOF
ADGUARD_USER={{ with secret "secret/data/adguard-home-gateway" }}{{ .Data.data.adguard_user }}{{ end }}
ADGUARD_PASSWORD={{ with secret "secret/data/adguard-home-gateway" }}{{ .Data.data.adguard_password }}{{ end }}
GATEWAY_API_KEY={{ with secret "secret/data/adguard-home-gateway" }}{{ .Data.data.api_key }}{{ end }}
EOF
        destination = "secrets/adguard-home-gateway.env"
        env         = true
      }

      env {
        ADGUARD_SERVERS         = "adguard1.example.com,adguard2.example.com"
        ADGUARD_SCHEME          = "http"
        ADGUARD_TLS_SKIP_VERIFY = "false"
        PORT                    = "8080"
        LOG_LEVEL               = "info"
      }

      resources {
        cpu    = 100
        memory = 64
      }

      kill_timeout = "35s"
    }
  }
}
