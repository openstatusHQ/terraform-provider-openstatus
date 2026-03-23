resource "openstatus_status_page_component" "api_monitor" {
  page_id    = openstatus_status_page.main.id
  type       = "monitor"
  monitor_id = openstatus_http_monitor.api.id
  name       = "API"
  order      = 1
}

resource "openstatus_status_page_component" "info" {
  page_id = openstatus_status_page.main.id
  type    = "static"
  name    = "Third-party Services"
  order   = 2
}
