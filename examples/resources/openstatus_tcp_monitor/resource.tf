resource "openstatus_tcp_monitor" "db" {
  name        = "Database Port Check"
  uri         = "db.example.com:5432"
  periodicity = "1m"
  timeout     = 10000
  active      = true
  regions     = ["fly-iad", "fly-fra"]

  open_telemetry {
    endpoint = "https://otel.example.com/v1/metrics"
  }
}
