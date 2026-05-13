# Changelog

## v0.2.0

### Breaking changes

- **`openstatus_notification.name` is now `Required`.** Configs that previously omitted `name` already failed at apply time against the API (server enforces `min_len=1`); this change surfaces the failure at plan time.
- **Unknown `access_type` values returned by the API are no longer silently coerced to `"public"`.** A status page in an unrecognized state will surface a diagnostic warning and leave `access_type` empty until a recognized value is set. The new `"ip"` value is also valid for `access_type`.
- **`description`, `body`, `timeout`, and `retry` are now sent on every monitor update,** including zero/empty values. The first apply after upgrade may clear a `description` that was set out-of-band (e.g., via the dashboard) but not declared in HCL. To avoid this, add `description = "..."` to HCL before upgrading.

### Added

- **New `ms_teams` notification provider** for Microsoft Teams webhooks.
- **Status page**: new `theme`, `default_locale`, `locales`, `allow_index`, and `allowed_ip_ranges` attributes; `theme` is now writable (was read-only).
- **Status page**: new `access_type = "ip"` mode (pairs with `allowed_ip_ranges`).
- **Monitors**: new `open_telemetry` block on `openstatus_http_monitor`, `openstatus_tcp_monitor`, and `openstatus_dns_monitor` to configure OpenTelemetry exporters.
- **Client-side slug regex validation** on `openstatus_status_page.slug` (matches the server's `^[a-z0-9]+(?:-[a-z0-9]+)*$`).
- **Diagnostic warnings** when the API returns unrecognized enum values (notification provider, Opsgenie region, status page theme/locale/access type) instead of silently corrupting state.

### Fixed

- **`openstatus_notification` updates to `monitor_ids` are now applied.** The provider now sends the API's `update_monitor_ids` flag and an explicit `monitor_ids` list (including empty `[]` to clear). Previously the field was silently ignored server-side.
- **Monitor `timeout = 0` and `retry = 0` are now honored.** Previously `omitempty` collapsed explicit zeros into "not set" and the server applied its defaults (45000ms / 3 retries).
