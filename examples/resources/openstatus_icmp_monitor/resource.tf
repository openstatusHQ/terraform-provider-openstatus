resource "openstatus_icmp_monitor" "gw" {
  name        = "Gateway Ping"
  uri         = "8.8.8.8"
  periodicity = "1m"
  timeout     = 10000
  active      = true
  regions     = ["fly-iad", "fly-fra"]

  open_telemetry {
    endpoint = "https://otel.example.com/v1/metrics"
  }
}