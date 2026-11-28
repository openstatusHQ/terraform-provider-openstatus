package playbook_test

import (
	"fmt"
	"regexp"
	"testing"

	"terraform-provider-openstatus/internal/testutil"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
)

func TestAccCustomPlaybook(t *testing.T) {
	server, fake := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				// The heredoc's trailing newline is trimmed by the API; the
				// apply must still be consistent and the next plan empty.
				Config: cfg + `
resource "openstatus_playbook" "vendor" {
  slug        = "vendor-outages"
  title       = "Vendor outages"
  description = "What to do when a third-party provider is down."
  content     = <<-EOT
    # Vendor outages

    Check {{vendor_status_url}} and tell {{oncall}}.
  EOT

  variables = {
    vendor_status_url = "https://status.example.com"
    oncall            = "@sre"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_playbook.vendor", "id", "vendor-outages"),
					resource.TestCheckResourceAttr("openstatus_playbook.vendor", "title", "Vendor outages"),
					resource.TestCheckResourceAttr("openstatus_playbook.vendor", "variables.oncall", "@sre"),
					resource.TestCheckResourceAttrSet("openstatus_playbook.vendor", "created_at"),
					func(*terraform.State) error {
						p := fake.Playbook("vendor-outages")
						if p == nil {
							return fmt.Errorf("playbook was not stored")
						}
						if want := "# Vendor outages\n\nCheck {{vendor_status_url}} and tell {{oncall}}."; p.GetContent() != want {
							return fmt.Errorf("stored content = %q, want %q", p.GetContent(), want)
						}
						return nil
					},
				),
			},
			{
				// Removing a variable unsets it on the server.
				Config: cfg + `
resource "openstatus_playbook" "vendor" {
  slug        = "vendor-outages"
  title       = "Vendor outages (renamed)"
  description = "What to do when a third-party provider is down."
  content     = "Tell {{oncall}}."

  variables = {
    oncall = "@platform"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_playbook.vendor", "title", "Vendor outages (renamed)"),
					resource.TestCheckResourceAttr("openstatus_playbook.vendor", "variables.%", "1"),
					resource.TestCheckResourceAttr("openstatus_playbook.vendor", "variables.oncall", "@platform"),
					func(*terraform.State) error {
						if vars := fake.Playbook("vendor-outages").GetVariables(); len(vars) != 1 || vars["oncall"] != "@platform" {
							return fmt.Errorf("stored variables = %v, want only oncall=@platform", vars)
						}
						return nil
					},
				),
			},
			{
				ResourceName:      "openstatus_playbook.vendor",
				ImportState:       true,
				ImportStateId:     "vendor-outages",
				ImportStateVerify: true,
			},
		},
	})

	if fake.Playbook("vendor-outages") != nil {
		t.Error("destroy did not delete the playbook")
	}
}

// Built-in slugs are written with UpdatePlaybook and default their title and
// description; destroying one reverts the agent to the OpenStatus default.
func TestAccBuiltinPlaybook(t *testing.T) {
	server, fake := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_playbook" "ir" {
  slug    = "incident-response"
  content = "SEV1: page {{oncall}} within 5 minutes."

  variables = {
    oncall = "@sre"
  }
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_playbook.ir", "title", "Incident response"),
					resource.TestCheckResourceAttrSet("openstatus_playbook.ir", "description"),
				),
			},
			{
				// Setting a title later overrides the default.
				Config: cfg + `
resource "openstatus_playbook" "ir" {
  slug    = "incident-response"
  title   = "How we run incidents"
  content = "SEV1: page {{oncall}} within 5 minutes."

  variables = {
    oncall = "@sre"
  }
}
`,
				Check: resource.TestCheckResourceAttr("openstatus_playbook.ir", "title", "How we run incidents"),
			},
			{
				// Moving to another built-in replaces the resource; its
				// default title must not be carried over from the old one.
				Config: cfg + `
resource "openstatus_playbook" "ir" {
  slug    = "postmortem"
  content = "Timeline, impact, root cause, follow-ups."
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_playbook.ir", "id", "postmortem"),
					resource.TestCheckResourceAttr("openstatus_playbook.ir", "title", "Postmortem"),
					resource.TestCheckNoResourceAttr("openstatus_playbook.ir", "variables"),
					func(*terraform.State) error {
						if fake.Playbook("incident-response") != nil {
							return fmt.Errorf("replacing the resource did not delete the old playbook")
						}
						return nil
					},
				),
			},
		},
	})
}

// A built-in the workspace already wrote must be imported, not overwritten:
// create would replace its content and a later destroy would delete it.
func TestAccBuiltinPlaybookAlreadyWritten(t *testing.T) {
	server, fake := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)
	fake.SeedPlaybook("postmortem", "Our postmortems", "Written in the dashboard.", "Hand-written template.")

	config := cfg + `
resource "openstatus_playbook" "pm" {
  slug    = "postmortem"
  content = "Hand-written template."
}
`
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config:      config,
				ExpectError: regexp.MustCompile(`Playbook already exists`),
			},
			{
				// After an import, the stored title is kept rather than
				// reset or pinned to the default.
				Config:             config,
				ResourceName:       "openstatus_playbook.pm",
				ImportState:        true,
				ImportStateId:      "postmortem",
				ImportStatePersist: true,
			},
			{
				Config: cfg + `
resource "openstatus_playbook" "pm" {
  slug    = "postmortem"
  content = "Updated template."
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("openstatus_playbook.pm", "title", "Our postmortems"),
					func(*terraform.State) error {
						p := fake.Playbook("postmortem")
						if p.GetContent() != "Updated template." || p.GetTitle() != "Our postmortems" {
							return fmt.Errorf("stored playbook = %q / %q, want updated content and the original title", p.GetTitle(), p.GetContent())
						}
						return nil
					},
				),
			},
		},
	})
}

func TestAccPlaybookValidation(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_playbook" "x" {
  slug    = "custom"
  content = "Hello."
}
`,
				ExpectError: regexp.MustCompile(`title is required`),
			},
			{
				Config: cfg + `
resource "openstatus_playbook" "x" {
  slug      = "status-page"
  content   = "No variables here."
  variables = { tone = "calm" }
}
`,
				ExpectError: regexp.MustCompile(`not referenced in content`),
			},
			{
				Config: cfg + `
resource "openstatus_playbook" "x" {
  slug    = "Not_A_Slug"
  content = "Hello."
}
`,
				ExpectError: regexp.MustCompile(`lowercase letters, digits and single hyphens`),
			},
		},
	})
}

func TestAccPlaybookDataSources(t *testing.T) {
	server, _ := testutil.NewServer(t)
	cfg := testutil.ProviderConfig(server)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testutil.ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: cfg + `
resource "openstatus_playbook" "sp" {
  slug    = "status-page"
  content = "Sign off as {{team}}."

  # Created after the custom playbook, so the list must reorder them.
  depends_on = [openstatus_playbook.custom]

  variables = {
    team = "The Acme team"
  }
}

resource "openstatus_playbook" "custom" {
  slug        = "vendor-outages"
  title       = "Vendor outages"
  description = "Third-party outages."
  content     = "Check the vendor."
}

data "openstatus_playbook" "sp" {
  slug = openstatus_playbook.sp.slug
}

data "openstatus_playbooks" "all" {
  depends_on = [openstatus_playbook.sp, openstatus_playbook.custom]
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.openstatus_playbook.sp", "title", "Status page"),
					resource.TestCheckResourceAttr("data.openstatus_playbook.sp", "content", "Sign off as {{team}}."),
					resource.TestCheckResourceAttr("data.openstatus_playbook.sp", "variables.team", "The Acme team"),
					resource.TestCheckResourceAttr("data.openstatus_playbooks.all", "playbooks.#", "2"),
					// Built-ins are listed ahead of custom playbooks.
					resource.TestCheckResourceAttr("data.openstatus_playbooks.all", "playbooks.0.slug", "status-page"),
					resource.TestCheckResourceAttr("data.openstatus_playbooks.all", "playbooks.1.slug", "vendor-outages"),
				),
			},
		},
	})
}
