data "openstatus_playbook" "status_page" {
  slug = "status-page"
}

output "status_page_playbook" {
  value = data.openstatus_playbook.status_page.content
}
