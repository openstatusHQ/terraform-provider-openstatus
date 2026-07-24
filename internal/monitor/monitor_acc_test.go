package monitor_test

import (
	"testing"

	"terraform-provider-openstatus/internal/testutil"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHTTPMonitor(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_http_monitor" "api" {
  name        = "API Health Check"
  url         = "https://api.example.com/health"
  periodicity = "5m"
  method      = "GET"
  timeout     = 30000
  active      = true
  regions     = ["fly-iad", "fly-ams"]

  headers {
    key   = "Authorization"
    value = "Bearer token"
  }

  status_code_assertions {
    target     = 200
    comparator = "eq"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("openstatus_http_monitor.api", "id"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "name", "API Health Check"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "url", "https://api.example.com/health"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "periodicity", "5m"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "timeout", "30000"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "regions.#", "2"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "headers.0.key", "Authorization"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "status_code_assertions.0.target", "200"),
				),
			},
			{
				Config: cfg + `
resource "openstatus_http_monitor" "api" {
  name        = "API Health Check renamed"
  url         = "https://api.example.com/healthz"
  periodicity = "1m"
  method      = "POST"
  timeout     = 45000
  active      = false
  regions     = ["fly-iad"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "name", "API Health Check renamed"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "periodicity", "1m"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "method", "POST"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "active", "false"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "regions.#", "1"),
					resource.TestCheckResourceAttr("openstatus_http_monitor.api", "headers.#", "0"),
				),
			},
		},
	})
}

func TestAccTCPMonitor(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_tcp_monitor" "db" {
  name        = "Database Port Check"
  uri         = "db.example.com:5432"
  periodicity = "1m"
  timeout     = 10000
  active      = true
  regions     = ["fly-iad", "fly-fra"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("openstatus_tcp_monitor.db", "id"),
					resource.TestCheckResourceAttr("openstatus_tcp_monitor.db", "uri", "db.example.com:5432"),
					resource.TestCheckResourceAttr("openstatus_tcp_monitor.db", "regions.#", "2"),
				),
			},
			{
				Config: cfg + `
resource "openstatus_tcp_monitor" "db" {
  name        = "Database Port Check v2"
  uri         = "db2.example.com:5432"
  periodicity = "5m"
  timeout     = 20000
  active      = true
  regions     = ["fly-iad"]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_tcp_monitor.db", "name", "Database Port Check v2"),
					resource.TestCheckResourceAttr("openstatus_tcp_monitor.db", "uri", "db2.example.com:5432"),
					resource.TestCheckResourceAttr("openstatus_tcp_monitor.db", "periodicity", "5m"),
				),
			},
		},
	})
}

func TestAccDNSMonitor(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_dns_monitor" "main" {
  name        = "DNS Resolution Check"
  uri         = "example.com"
  periodicity = "10m"
  active      = true
  regions     = ["fly-iad"]

  record_assertions {
    record     = "A"
    comparator = "eq"
    target     = "93.184.216.34"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("openstatus_dns_monitor.main", "id"),
					resource.TestCheckResourceAttr("openstatus_dns_monitor.main", "uri", "example.com"),
					resource.TestCheckResourceAttr("openstatus_dns_monitor.main", "record_assertions.0.record", "A"),
					resource.TestCheckResourceAttr("openstatus_dns_monitor.main", "record_assertions.0.target", "93.184.216.34"),
				),
			},
			{
				Config: cfg + `
resource "openstatus_dns_monitor" "main" {
  name        = "DNS Resolution Check v2"
  uri         = "example.org"
  periodicity = "30m"
  active      = true
  regions     = ["fly-iad", "fly-ams"]

  record_assertions {
    record     = "CNAME"
    comparator = "eq"
    target     = "alias.example.org"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_dns_monitor.main", "uri", "example.org"),
					resource.TestCheckResourceAttr("openstatus_dns_monitor.main", "periodicity", "30m"),
					resource.TestCheckResourceAttr("openstatus_dns_monitor.main", "record_assertions.0.record", "CNAME"),
				),
			},
		},
	})
}

func TestAccMonitorDataSources(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_http_monitor" "api" {
  name        = "Data Source Probe"
  url         = "https://api.example.com/health"
  periodicity = "5m"
  regions     = ["fly-iad"]
}

data "openstatus_monitor" "existing" {
  id = openstatus_http_monitor.api.id
}

data "openstatus_monitors" "all" {
  limit      = 100
  depends_on = [openstatus_http_monitor.api]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.openstatus_monitor.existing", "name", "Data Source Probe"),
					resource.TestCheckResourceAttr("data.openstatus_monitor.existing", "url", "https://api.example.com/health"),
					resource.TestCheckResourceAttr("data.openstatus_monitors.all", "monitors.#", "1"),
					resource.TestCheckResourceAttr("data.openstatus_monitors.all", "monitors.0.name", "Data Source Probe"),
					resource.TestCheckResourceAttr("data.openstatus_monitors.all", "monitors.0.type", "http"),
				),
			},
		},
	})
}
