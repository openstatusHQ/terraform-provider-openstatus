data "openstatus_private_locations" "all" {
  limit = 100
}

# Agent tokens are not returned by the list endpoint; use the
# openstatus_private_location data source to fetch one by ID.
output "private_location_names" {
  value = [for l in data.openstatus_private_locations.all.private_locations : l.name]
}
