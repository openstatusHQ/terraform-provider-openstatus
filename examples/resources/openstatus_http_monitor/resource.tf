resource "openstatus_http_monitor" "api" {
  name        = "API Health Check"
  url         = "https://api.example.com/health"
  periodicity = "5m"
  method      = "GET"
  timeout     = 30000
  active      = true
  regions     = ["fly-iad", "fly-ams", "fly-syd"]

  headers {
    key   = "Authorization"
    value = "Bearer token"
  }

  status_code_assertions {
    target     = 200
    comparator = "eq"
  }

  body_assertions {
    target     = "ok"
    comparator = "contains"
  }
}
