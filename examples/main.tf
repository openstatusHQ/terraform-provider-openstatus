terraform {
  required_providers {
    openstatus = {
      source = "openstatus.dev/tf/openstatus"

    }
  }
}

provider "openstatus" {
  openstatus_api_token = "your-key"
}

resource "openstatus_monitor" "my_monitor" {
  url            = "https://www.openstatus.dev"
  regions        = ["iad", "jnb", "ams"]
  periodicity    = "10m"
  name           = "test-monitor"
  degraded_after = 30
  timeout        = 13
  active         = false
  description    = "This is a test monitor"
  headers = [
    { key = "test-key", value = "test-value" },
  ]
  assertions = [
    {
      type    = "status"
      target  = "200"
      key     = ""
      compare = "eq"
    },
    {
      type    = "status"
      target  = "300"
      key     = ""
      compare = "eq"
    },
    {
      type    = "header"
      target  = "test"
      key     = "test"
      compare = "eq"
    }
  ]
}
