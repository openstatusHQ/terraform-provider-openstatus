data "openstatus_status_page" "main" {
  id = "456"
}

output "status_page_slug" {
  value = data.openstatus_status_page.main.slug
}
