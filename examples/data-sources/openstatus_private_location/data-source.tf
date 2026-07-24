data "openstatus_private_location" "eu_dc" {
  id = "pl_123"
}

output "private_location_status" {
  value = data.openstatus_private_location.eu_dc.status
}

output "private_location_last_seen_at" {
  value = data.openstatus_private_location.eu_dc.last_seen_at
}
