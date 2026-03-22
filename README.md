# Terraform Provider OpenStatus

Terraform provider for managing [OpenStatus](https://openstatus.dev) resources: monitors, notifications, and status pages.

## Usage

```hcl
terraform {
  required_providers {
    openstatus = {
      source = "openstatusHQ/openstatus"
    }
  }
}

provider "openstatus" {
  # Set via OPENSTATUS_API_TOKEN environment variable or:
  # api_token = "your-token"
}

resource "openstatus_http_monitor" "api" {
  name        = "API Health"
  url         = "https://api.example.com/health"
  periodicity = "5m"
  active      = true

  status_code_assertions {
    target     = 200
    comparator = "eq"
  }
}
```

## Resources

- `openstatus_http_monitor` — HTTP monitors with assertions
- `openstatus_tcp_monitor` — TCP connection monitors
- `openstatus_dns_monitor` — DNS record monitors
- `openstatus_notification` — Notification channels (Slack, Discord, PagerDuty, email, webhook, etc.)
- `openstatus_status_page` — Status pages
- `openstatus_status_page_component` — Status page components (monitor or static)
- `openstatus_status_page_component_group` — Component groups

## Data Sources

- `openstatus_monitor` — Look up a monitor by ID
- `openstatus_monitors` — List all monitors
- `openstatus_status_page` — Look up a status page by ID
- `openstatus_notification` — Look up a notification by ID

## Development

```sh
make build      # build the provider
make test       # run unit tests
make testacc    # run acceptance tests (requires OPENSTATUS_API_TOKEN)
make install    # install to local plugin directory
```
