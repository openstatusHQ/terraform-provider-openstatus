package statuspage_test

import (
	"testing"

	"terraform-provider-openstatus/internal/testutil"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccStatusPage(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_status_page" "main" {
  title          = "Example Status Page"
  slug           = "example-status"
  description    = "Status page for Example Inc."
  homepage_url   = "https://example.com"
  theme          = "dark"
  default_locale = "en"
  locales        = ["en", "fr"]
  allow_index    = true
}

data "openstatus_status_page" "main" {
  id = openstatus_status_page.main.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("openstatus_status_page.main", "id"),
					resource.TestCheckResourceAttr("openstatus_status_page.main", "title", "Example Status Page"),
					resource.TestCheckResourceAttr("openstatus_status_page.main", "slug", "example-status"),
					resource.TestCheckResourceAttr("openstatus_status_page.main", "theme", "dark"),
					resource.TestCheckResourceAttr("openstatus_status_page.main", "locales.#", "2"),
					resource.TestCheckResourceAttr("data.openstatus_status_page.main", "title", "Example Status Page"),
					resource.TestCheckResourceAttr("data.openstatus_status_page.main", "slug", "example-status"),
				),
			},
			{
				Config: cfg + `
resource "openstatus_status_page" "main" {
  title          = "Renamed Status Page"
  slug           = "example-status"
  description    = "Updated description"
  theme          = "light"
  default_locale = "en"
  locales        = ["en"]
  allow_index    = false
}

data "openstatus_status_page" "main" {
  id = openstatus_status_page.main.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_status_page.main", "title", "Renamed Status Page"),
					resource.TestCheckResourceAttr("openstatus_status_page.main", "description", "Updated description"),
					resource.TestCheckResourceAttr("openstatus_status_page.main", "theme", "light"),
					resource.TestCheckResourceAttr("openstatus_status_page.main", "locales.#", "1"),
					resource.TestCheckResourceAttr("openstatus_status_page.main", "allow_index", "false"),
				),
			},
		},
	})
}

func TestAccStatusPageCustomTheme(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_status_page" "themed" {
  title = "Themed Page"
  slug  = "themed-page"

  custom_theme = {
    light = {
      "--primary" = "hsl(24 94% 50%)"
      "--radius"  = "0.5rem"
    }
    dark = {
      "--primary" = "hsl(24 94% 60%)"
    }
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_status_page.themed", "custom_theme.light.--primary", "hsl(24 94% 50%)"),
					resource.TestCheckResourceAttr("openstatus_status_page.themed", "custom_theme.light.--radius", "0.5rem"),
					resource.TestCheckResourceAttr("openstatus_status_page.themed", "custom_theme.dark.--primary", "hsl(24 94% 60%)"),
				),
			},
			{
				Config: cfg + `
resource "openstatus_status_page" "themed" {
  title = "Themed Page"
  slug  = "themed-page"
}
`,
				Check: resource.TestCheckNoResourceAttr("openstatus_status_page.themed", "custom_theme.light.--primary"),
			},
		},
	})
}

func TestAccStatusPageComponentAndGroup(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_status_page" "main" {
  title = "Components Page"
  slug  = "components-page"
}

resource "openstatus_status_page_component_group" "infra" {
  page_id      = openstatus_status_page.main.id
  name         = "Infrastructure"
  default_open = true
}

resource "openstatus_http_monitor" "api" {
  name        = "Component Monitor"
  url         = "https://api.example.com/health"
  periodicity = "5m"
  regions     = ["fly-iad"]
}

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
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("openstatus_status_page_component_group.infra", "id"),
					resource.TestCheckResourceAttr("openstatus_status_page_component_group.infra", "name", "Infrastructure"),
					resource.TestCheckResourceAttr("openstatus_status_page_component_group.infra", "default_open", "true"),
					resource.TestCheckResourceAttr("openstatus_status_page_component.api_monitor", "type", "monitor"),
					resource.TestCheckResourceAttr("openstatus_status_page_component.api_monitor", "name", "API"),
					resource.TestCheckResourceAttr("openstatus_status_page_component.api_monitor", "order", "1"),
					resource.TestCheckResourceAttr("openstatus_status_page_component.info", "type", "static"),
					resource.TestCheckResourceAttr("openstatus_status_page_component.info", "order", "2"),
				),
			},
			{
				Config: cfg + `
resource "openstatus_status_page" "main" {
  title = "Components Page"
  slug  = "components-page"
}

resource "openstatus_status_page_component_group" "infra" {
  page_id      = openstatus_status_page.main.id
  name         = "Core Infrastructure"
  default_open = false
}

resource "openstatus_http_monitor" "api" {
  name        = "Component Monitor"
  url         = "https://api.example.com/health"
  periodicity = "5m"
  regions     = ["fly-iad"]
}

resource "openstatus_status_page_component" "api_monitor" {
  page_id    = openstatus_status_page.main.id
  type       = "monitor"
  monitor_id = openstatus_http_monitor.api.id
  name       = "API Gateway"
  order      = 3
}

resource "openstatus_status_page_component" "info" {
  page_id = openstatus_status_page.main.id
  type    = "static"
  name    = "Third-party Services"
  order   = 2
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_status_page_component_group.infra", "name", "Core Infrastructure"),
					resource.TestCheckResourceAttr("openstatus_status_page_component_group.infra", "default_open", "false"),
					resource.TestCheckResourceAttr("openstatus_status_page_component.api_monitor", "name", "API Gateway"),
					resource.TestCheckResourceAttr("openstatus_status_page_component.api_monitor", "order", "3"),
				),
			},
		},
	})
}
