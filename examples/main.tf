terraform {
  required_providers {
    openstatus = {
      source = "openstatus/openstatus"
    }
  }
}

provider "openstatus" {
  openstatus_api_token= "YOUR_API_TOKEN"
}

resource "openstatus_monitor" "my_monitor" {
  url   = "https://www.openstatus.dev"
  regions= ["iad", "jnb"]
  periodicity =  "10m"
  name = "test-monitor"
  active = true
  description = "This is a test monitor"
}