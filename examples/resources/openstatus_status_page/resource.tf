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

  # Per-mode CSS variable overrides (requires the custom-theme plan feature).
  custom_theme = {
    light = {
      "--primary" = "hsl(24 94% 50%)"
      "--radius"  = "0.5rem"
    }
    dark = {
      "--primary"    = "hsl(24 94% 60%)"
      "--background" = "hsl(240 10% 4%)"
    }
  }
}

resource "openstatus_status_page" "internal" {
  title             = "Internal Status"
  slug              = "internal-status"
  access_type       = "ip"
  allowed_ip_ranges = "10.0.0.0/8,192.168.0.0/16"
}
