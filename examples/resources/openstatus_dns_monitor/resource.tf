resource "openstatus_dns_monitor" "main" {
  name        = "DNS Resolution Check"
  uri         = "example.com"
  periodicity = "10m"
  active      = true
  regions     = ["fly-iad"]

  record_assertions {
    record     = "A"
    comparator = "eq"
    target     = "93.184.216.34"
  }

  open_telemetry {
    endpoint = "https://otel.example.com/v1/metrics"
  }
}
