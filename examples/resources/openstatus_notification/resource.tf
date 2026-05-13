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

resource "openstatus_notification" "teams" {
  name          = "Teams Alerts"
  provider_type = "ms_teams"

  ms_teams {
    webhook_url = "https://prod-00.westeurope.logic.azure.com:443/workflows/abc/triggers/manual/paths/invoke"
  }
}
