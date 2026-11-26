package statuspage_test

import (
	"fmt"
	"regexp"
	"testing"

	"terraform-provider-openstatus/internal/testutil"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

const publicBaseConfig = `
resource "openstatus_status_page" "main" {
  title = "Visibility Page"
  slug  = "visibility-page"
}

resource "openstatus_http_monitor" "api" {
  name        = "Visibility Monitor"
  url         = "https://api.example.com/health"
  periodicity = "5m"
  regions     = ["fly-iad"]
}
`

func monitorComponentConfig(publicLine string) string {
	return publicBaseConfig + fmt.Sprintf(`
resource "openstatus_status_page_component" "api" {
  page_id    = openstatus_status_page.main.id
  type       = "monitor"
  monitor_id = openstatus_http_monitor.api.id
  name       = "API"
  %s
}
`, publicLine)
}

func TestAccStatusPageComponentPublic(t *testing.T) {
	server, fake := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)
	const name = "openstatus_status_page_component.api"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + monitorComponentConfig("public = true"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "public", "true"),
					checkServerPublic(fake, name, true),
				),
			},
			{Config: cfg + monitorComponentConfig("public = true"), PlanOnly: true},
			{
				// false must actually be sent; proto3 would drop it without presence.
				Config: cfg + monitorComponentConfig("public = false"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "public", "false"),
					checkServerPublic(fake, name, false),
				),
			},
			{Config: cfg + monitorComponentConfig("public = false"), PlanOnly: true},
		},
	})
}

// After the API migration, existing components can come back public = true
// while their config doesn't mention `public`. That must not plan a change.
func TestAccStatusPageComponentPublicUnsetKeepsServerValue(t *testing.T) {
	server, fake := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)
	const name = "openstatus_status_page_component.api"
	var componentID string

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + monitorComponentConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "public", "false"),
					func(s *terraform.State) error {
						componentID = s.RootModule().Resources[name].Primary.ID
						return nil
					},
				),
			},
			{
				PreConfig: func() { fake.Component(componentID).SetPublic(true) },
				Config:    cfg + monitorComponentConfig(""),
				PlanOnly:  true,
			},
			{
				Config: cfg + monitorComponentConfig(""),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr(name, "public", "true"),
					checkServerPublic(fake, name, true),
				),
			},
		},
	})
}

func TestAccStatusPageComponentPublicRejectedOnStatic(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + publicBaseConfig + `
resource "openstatus_status_page_component" "info" {
  page_id = openstatus_status_page.main.id
  type    = "static"
  name    = "Third-party Services"
  public  = true
}
`,
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`Static components cannot be public`),
			},
		},
	})
}

func TestAccStatusPageComponentPublicRejectedOnHeartbeat(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + publicBaseConfig + `
resource "openstatus_heartbeat_monitor" "backup" {
  name   = "Nightly backup"
  period = 86400
  grace  = 1800
}

resource "openstatus_status_page_component" "backup" {
  page_id    = openstatus_status_page.main.id
  type       = "monitor"
  monitor_id = openstatus_heartbeat_monitor.backup.id
  public     = true
}
`,
				ExpectError: regexp.MustCompile(`heartbeat monitors cannot be public`),
			},
		},
	})
}

func checkServerPublic(fake *testutil.Fake, resourceName string, want bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("%s not found in state", resourceName)
		}
		component := fake.Component(rs.Primary.ID)
		if component == nil {
			return fmt.Errorf("component %s does not exist on the server", rs.Primary.ID)
		}
		if got := component.GetPublic(); got != want {
			return fmt.Errorf("server-side public = %v, want %v", got, want)
		}
		return nil
	}
}
