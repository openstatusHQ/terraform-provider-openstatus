resource "openstatus_grpc_monitor" "api" {
  name        = "API gRPC Health"
  uri         = "grpc.example.com:443"
  periodicity = "1m"
  service     = "my.package.Service"
  tls_mode    = "tls"
  active      = true
  regions     = ["fly-iad", "fly-fra"]

  metadata {
    key   = "authorization"
    value = "Bearer ${var.grpc_token}"
  }
}
