resource "openstatus_status_page_component_group" "infrastructure" {
  page_id      = openstatus_status_page.main.id
  name         = "Infrastructure"
  default_open = true
}
