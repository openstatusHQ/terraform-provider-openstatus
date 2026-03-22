resource "openstatus_notification" "slack" {
  name          = "Slack Alerts"
  provider_type = "slack"
  monitor_ids   = [openstatus_http_monitor.api.id]

  slack {
    webhook_url = "https://hooks.slack.com/services/xxx/yyy/zzz"
  }
}

resource "openstatus_notification" "email" {
  name          = "Email Alerts"
  provider_type = "email"

  email {
    email = "oncall@example.com"
  }
}
