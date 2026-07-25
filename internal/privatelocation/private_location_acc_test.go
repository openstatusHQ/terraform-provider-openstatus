package privatelocation_test

import (
	"testing"

	"terraform-provider-openstatus/internal/testutil"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccPrivateLocation(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_private_location" "eu" {
  name = "EU Datacenter"

  metadata = {
    env    = "prod"
    region = "eu-west-1"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("openstatus_private_location.eu", "id"),
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "name", "EU Datacenter"),
					resource.TestCheckResourceAttrSet("openstatus_private_location.eu", "token"),
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "metadata.env", "prod"),
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "metadata.region", "eu-west-1"),
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "monitor_ids.#", "0"),
				),
			},
			{
				Config: cfg + `
resource "openstatus_private_location" "eu" {
  name = "EU Datacenter renamed"

  metadata = {
    env = "staging"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "name", "EU Datacenter renamed"),
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "metadata.env", "staging"),
					resource.TestCheckNoResourceAttr("openstatus_private_location.eu", "metadata.region"),
				),
			},
			{
				ResourceName:      "openstatus_private_location.eu",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Removing monitor_ids and metadata from the configuration must clear them,
// which only works because update always sends the sentinels.
func TestAccPrivateLocationClearsAssociations(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_http_monitor" "api" {
  name        = "Private Monitor"
  url         = "https://api.example.com/health"
  periodicity = "5m"
  regions     = ["fly-iad"]
}

resource "openstatus_private_location" "eu" {
  name        = "EU Datacenter"
  monitor_ids = [openstatus_http_monitor.api.id]

  metadata = {
    env = "prod"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "monitor_ids.#", "1"),
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "metadata.env", "prod"),
				),
			},
			{
				Config: cfg + `
resource "openstatus_http_monitor" "api" {
  name        = "Private Monitor"
  url         = "https://api.example.com/health"
  periodicity = "5m"
  regions     = ["fly-iad"]
}

resource "openstatus_private_location" "eu" {
  name = "EU Datacenter"
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "monitor_ids.#", "0"),
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "metadata.%", "0"),
				),
			},
		},
	})
}

// `monitor_ids = []` and `metadata = {}` are not the same as omitting them:
// the plan holds an empty collection, so post-apply state must too.
func TestAccPrivateLocationExplicitEmptyCollections(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	config := cfg + `
resource "openstatus_private_location" "eu" {
  name        = "EU Datacenter"
  monitor_ids = []
  metadata    = {}
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "monitor_ids.#", "0"),
					resource.TestCheckResourceAttr("openstatus_private_location.eu", "metadata.%", "0"),
				),
			},
			// Refresh must not flip the empty collections back to null.
			{Config: config, PlanOnly: true},
		},
	})
}

func TestAccPrivateLocationDataSources(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_private_location" "eu" {
  name = "EU Datacenter"

  metadata = {
    env = "prod"
  }
}

data "openstatus_private_location" "eu" {
  id = openstatus_private_location.eu.id
}

data "openstatus_private_locations" "all" {
  depends_on = [openstatus_private_location.eu]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.openstatus_private_location.eu", "name", "EU Datacenter"),
					resource.TestCheckResourceAttr("data.openstatus_private_location.eu", "status", "active"),
					resource.TestCheckResourceAttrSet("data.openstatus_private_location.eu", "token"),
					resource.TestCheckResourceAttr("data.openstatus_private_location.eu", "metadata.env", "prod"),
					resource.TestCheckResourceAttr("data.openstatus_private_locations.all", "private_locations.#", "1"),
					resource.TestCheckResourceAttr("data.openstatus_private_locations.all", "private_locations.0.name", "EU Datacenter"),
					resource.TestCheckResourceAttr("data.openstatus_private_locations.all", "private_locations.0.monitor_count", "0"),
					resource.TestCheckResourceAttr("data.openstatus_private_locations.all", "private_locations.0.status", "active"),
				),
			},
		},
	})
}
