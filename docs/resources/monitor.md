---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "openstatus_monitor Resource - terraform-provider-openstatus"
subcategory: ""
description: |-

---

# openstatus_monitor (Resource)

<https://docs.openstatus.dev/synthetic/features/monitor>



<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `name` (String) The name of the monitor
- `periodicity` (String) How often the monitor should run
- `url` (String) The url to monitor

### Optional

- `active` (Boolean) If the monitor is active
- `assertions` (Attributes List) The assertions to run (see [below for nested schema](#nestedatt--assertions))
- `body` (String) The body
- `degraded_after` (Number) The time after the monitor is considered degraded
- `description` (String) The description of your monitor
- `headers` (Attributes List) The headers of your request (see [below for nested schema](#nestedatt--headers))
- `id` (Number) The id of the monitor
- `method` (String)
- `public` (Boolean) If the monitor is public
- `regions` (List of String) Where we should monitor it
- `timeout` (Number) The timeout of the request

<a id="nestedatt--assertions"></a>
### Nested Schema for `assertions`

Required:

- `compare` (String) The comparison to run
- `target` (String) The target value
- `type` (String)

Optional:

- `key` (String) The key to check


<a id="nestedatt--headers"></a>
### Nested Schema for `headers`

Required:

- `key` (String)
- `value` (String)
