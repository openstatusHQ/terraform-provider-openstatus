# Override a built-in playbook. Title and description default to the
# OpenStatus ones; destroying the resource reverts to the OpenStatus default.
# If the workspace already wrote this playbook, import it first (see below).
resource "openstatus_playbook" "incident_response" {
  slug    = "incident-response"
  content = <<-EOT
    # Incident response

    - SEV1: customer-facing outage. Page {{oncall}} immediately.
    - SEV2: degraded service. Post in {{incident_channel}} within 15 minutes.
    - Update the status page every 30 minutes until resolved.
  EOT

  variables = {
    oncall           = "@sre-oncall"
    incident_channel = "#incidents"
  }
}

# A custom playbook needs a title and a description. The agent reads the
# description to decide when to open the playbook.
resource "openstatus_playbook" "vendor_outages" {
  slug        = "vendor-outages"
  title       = "Vendor outages"
  description = "What to do when a third-party provider (payments, email, CDN) is down."
  content     = file("${path.module}/playbooks/vendor-outages.md")
}
