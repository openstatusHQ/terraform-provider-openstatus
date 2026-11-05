package monitor_test

import (
	"regexp"
	"testing"

	"terraform-provider-openstatus/internal/testutil"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccMonitors_RejectDeprecatedRegion(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	for resourceType, target := range map[string]string{
		"openstatus_http_monitor": `url = "https://example.com"`,
		"openstatus_tcp_monitor":  `uri = "example.com:443"`,
		"openstatus_dns_monitor":  `uri = "example.com"`,
		"openstatus_icmp_monitor": `uri = "example.com"`,
		"openstatus_grpc_monitor": `uri = "example.com:443"`,
	} {
		t.Run(resourceType, func(t *testing.T) {
			resource.Test(t, resource.TestCase{
				ProtoV6ProviderFactories: testutil.ProviderFactories(),
				Steps: []resource.TestStep{{
					Config: cfg + `
resource "` + resourceType + `" "m" {
  name        = "m"
  ` + target + `
  periodicity = "5m"
  regions     = ["fly-bom"]
}
`,
					PlanOnly:    true,
					ExpectError: regexp.MustCompile(`regions\[Value\("fly-bom"\)\]`),
				}},
			})
		})
	}
}
