package monitor_test

import (
	"regexp"
	"testing"

	"terraform-provider-openstatus/internal/testutil"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccHeartbeatMonitor(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)
	const pingURL = "https://api.openstatus.dev/heartbeat/token-heartbeat_1"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_heartbeat_monitor" "backup" {
  name        = "Nightly backup"
  description = "Dumps the database"
  period      = 86400
  grace       = 1800
  active      = true
  public      = true
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("openstatus_heartbeat_monitor.backup", "id"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "name", "Nightly backup"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "description", "Dumps the database"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "period", "86400"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "grace", "1800"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "active", "true"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "public", "true"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "ping_url", pingURL),
				),
			},
			{
				// Removing the optional fields must clear them on the server,
				// and the ping URL must survive an update.
				Config: cfg + `
resource "openstatus_heartbeat_monitor" "backup" {
  name   = "Hourly backup"
  period = 3600
  grace  = 300
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "name", "Hourly backup"),
					resource.TestCheckNoResourceAttr("openstatus_heartbeat_monitor.backup", "description"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "period", "3600"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "grace", "300"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "active", "false"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "public", "false"),
					resource.TestCheckResourceAttr("openstatus_heartbeat_monitor.backup", "ping_url", pingURL),
				),
			},
			{
				ResourceName:      "openstatus_heartbeat_monitor.backup",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccHeartbeatMonitor_Bounds(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	config := func(period, grace string) string {
		return cfg + `
resource "openstatus_heartbeat_monitor" "backup" {
  name   = "Backup"
  period = ` + period + `
  grace  = ` + grace + `
}
`
	}

	steps := []resource.TestStep{}
	for _, tc := range []struct{ period, grace, want string }{
		{"59", "60", "period"},
		{"2592001", "60", "period"},
		{"60", "59", "grace"},
		{"60", "604801", "grace"},
	} {
		steps = append(steps, resource.TestStep{
			Config:      config(tc.period, tc.grace),
			ExpectError: regexp.MustCompile(`(?i)Invalid Attribute Value[\s\S]*` + tc.want),
		})
	}
	// The bounds themselves are valid.
	steps = append(steps,
		resource.TestStep{Config: config("60", "60")},
		resource.TestStep{Config: config("2592000", "604800")},
	)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps:                    steps,
	})
}

// A ping changes the monitor's status and last ping out of band. The data
// source must report them, and the resource must not plan a change because of
// them: the framework fails a step when the post-apply plan is not empty.
func TestAccHeartbeatMonitor_PingIsNotDrift(t *testing.T) {
	server, fake := testutil.NewServer(t)
	config := testutil.ProviderConfig(server) + `
resource "openstatus_heartbeat_monitor" "backup" {
  name   = "Backup"
  period = 3600
  grace  = 300
  active = true
}

data "openstatus_monitor" "backup" {
  id = openstatus_heartbeat_monitor.backup.id
}

data "openstatus_monitors" "all" {
  depends_on = [openstatus_heartbeat_monitor.backup]
}
`

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.openstatus_monitor.backup", "type", "heartbeat"),
					resource.TestCheckResourceAttr("data.openstatus_monitor.backup", "name", "Backup"),
					resource.TestCheckResourceAttr("data.openstatus_monitor.backup", "period", "3600"),
					resource.TestCheckResourceAttr("data.openstatus_monitor.backup", "grace", "300"),
					resource.TestCheckResourceAttrPair("data.openstatus_monitor.backup", "ping_url", "openstatus_heartbeat_monitor.backup", "ping_url"),
					resource.TestCheckResourceAttr("data.openstatus_monitor.backup", "status", "active"),
					resource.TestCheckNoResourceAttr("data.openstatus_monitor.backup", "last_ping_at"),
					resource.TestCheckNoResourceAttr("data.openstatus_monitor.backup", "last_ping_kind"),
					resource.TestCheckResourceAttr("data.openstatus_monitors.all", "monitors.#", "1"),
					resource.TestCheckResourceAttr("data.openstatus_monitors.all", "monitors.0.type", "heartbeat"),
					resource.TestCheckResourceAttr("data.openstatus_monitors.all", "monitors.0.name", "Backup"),
				),
			},
			{
				PreConfig: func() {
					fake.RecordHeartbeatPing(
						"heartbeat_1",
						"2026-11-05T10:00:00.000Z",
						monitorv1.HeartbeatPingKind_HEARTBEAT_PING_KIND_FAIL,
						monitorv1.MonitorStatus_MONITOR_STATUS_ERROR,
					)
				},
				Config: config,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.openstatus_monitor.backup", "status", "error"),
					resource.TestCheckResourceAttr("data.openstatus_monitor.backup", "last_ping_at", "2026-11-05T10:00:00.000Z"),
					resource.TestCheckResourceAttr("data.openstatus_monitor.backup", "last_ping_kind", "fail"),
				),
			},
		},
	})
}
