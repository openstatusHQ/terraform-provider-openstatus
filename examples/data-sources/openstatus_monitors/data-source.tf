data "openstatus_monitors" "all" {
  limit = 100
}

output "monitor_count" {
  value = length(data.openstatus_monitors.all.monitors)
}
