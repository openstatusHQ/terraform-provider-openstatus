resource "openstatus_status_page" "main" {
  title          = "Example Status Page"
  slug           = "example-status"
  description    = "Status page for Example Inc."
  homepage_url   = "https://example.com"
  contact_url    = "https://example.com/contact"
  theme          = "dark"
  default_locale = "en"
  locales        = ["en", "fr"]
  allow_index    = true
}

resource "openstatus_status_page" "internal" {
  title             = "Internal Status"
  slug              = "internal-status"
  access_type       = "ip"
  allowed_ip_ranges = "10.0.0.0/8,192.168.0.0/16"
}
