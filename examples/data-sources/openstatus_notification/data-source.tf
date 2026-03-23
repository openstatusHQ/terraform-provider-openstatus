data "openstatus_notification" "slack" {
  id = "789"
}

output "notification_provider" {
  value = data.openstatus_notification.slack.provider_type
}
