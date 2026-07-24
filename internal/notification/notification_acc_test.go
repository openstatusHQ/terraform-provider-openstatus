package notification_test

import (
	"testing"

	"terraform-provider-openstatus/internal/testutil"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccNotification(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_notification" "slack" {
  name          = "Slack Alerts"
  provider_type = "slack"

  slack {
    webhook_url = "https://hooks.slack.com/services/xxx/yyy/zzz"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("openstatus_notification.slack", "id"),
					resource.TestCheckResourceAttr("openstatus_notification.slack", "name", "Slack Alerts"),
					resource.TestCheckResourceAttr("openstatus_notification.slack", "provider_type", "slack"),
					resource.TestCheckResourceAttr("openstatus_notification.slack", "slack.0.webhook_url", "https://hooks.slack.com/services/xxx/yyy/zzz"),
				),
			},
			{
				Config: cfg + `
resource "openstatus_notification" "slack" {
  name          = "Slack Alerts renamed"
  provider_type = "slack"

  slack {
    webhook_url = "https://hooks.slack.com/services/aaa/bbb/ccc"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_notification.slack", "name", "Slack Alerts renamed"),
					resource.TestCheckResourceAttr("openstatus_notification.slack", "slack.0.webhook_url", "https://hooks.slack.com/services/aaa/bbb/ccc"),
				),
			},
		},
	})
}

// The API never echoes ntfy.token back, so the provider must preserve the
// planned value rather than emptying it on read-back.
func TestAccNotificationNtfyTokenPreserved(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_notification" "ntfy" {
  name          = "Ntfy Alerts"
  provider_type = "ntfy"

  ntfy {
    topic      = "alerts"
    server_url = "https://ntfy.example.com"
    token      = "tk_secret"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_notification.ntfy", "ntfy.0.topic", "alerts"),
					resource.TestCheckResourceAttr("openstatus_notification.ntfy", "ntfy.0.server_url", "https://ntfy.example.com"),
					resource.TestCheckResourceAttr("openstatus_notification.ntfy", "ntfy.0.token", "tk_secret"),
				),
			},
		},
	})
}

func TestAccNotificationWithMonitors(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_http_monitor" "api" {
  name        = "Notified Monitor"
  url         = "https://api.example.com/health"
  periodicity = "5m"
  regions     = ["fly-iad"]
}

resource "openstatus_notification" "email" {
  name          = "Email Alerts"
  provider_type = "email"
  monitor_ids   = [openstatus_http_monitor.api.id]

  email {
    email = "oncall@example.com"
  }
}

data "openstatus_notification" "email" {
  id = openstatus_notification.email.id
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_notification.email", "monitor_ids.#", "1"),
					resource.TestCheckResourceAttr("data.openstatus_notification.email", "name", "Email Alerts"),
					resource.TestCheckResourceAttr("data.openstatus_notification.email", "provider_type", "email"),
				),
			},
			{
				Config: cfg + `
resource "openstatus_http_monitor" "api" {
  name        = "Notified Monitor"
  url         = "https://api.example.com/health"
  periodicity = "5m"
  regions     = ["fly-iad"]
}

resource "openstatus_notification" "email" {
  name          = "Email Alerts"
  provider_type = "email"

  email {
    email = "oncall@example.com"
  }
}

data "openstatus_notification" "email" {
  id = openstatus_notification.email.id
}
`,
				Check: resource.TestCheckResourceAttr("openstatus_notification.email", "monitor_ids.#", "0"),
			},
		},
	})
}
