data "openstatus_playbooks" "all" {}

# Content is not returned by the list endpoint; use the openstatus_playbook
# data source to fetch one by slug.
output "playbook_slugs" {
  value = [for p in data.openstatus_playbooks.all.playbooks : p.slug]
}
