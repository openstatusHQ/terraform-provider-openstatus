data "openstatus_monitor" "existing" {
  id = "123"
}

output "monitor_name" {
  value = data.openstatus_monitor.existing.name
}
