package playbook

import (
	"context"
	"strings"
	"testing"

	playbookv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/playbook/v1"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestVariableNames(t *testing.T) {
	got := VariableNames("Page {{ oncall }} then {{team_name}}, again {{oncall}}. Not {{Upper}} or {{a-b}}.")
	want := []string{"oncall", "team_name"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("VariableNames = %v, want %v", got, want)
	}
}

func TestValidateContent(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		variables map[string]string
		wantErr   string
	}{
		{name: "valid", content: "Page {{oncall}}.", variables: map[string]string{"oncall": "@sre"}},
		{name: "padded reference", content: "Page {{ oncall }}.", variables: map[string]string{"oncall": "@sre"}},
		{name: "reference without value", content: "Page {{oncall}}."},
		{name: "whitespace only", content: "  \n\t", wantErr: "cannot be empty"},
		{name: "unused variable", content: "No refs.", variables: map[string]string{"oncall": "@sre"}, wantErr: "not referenced"},
		{name: "empty value", content: "{{oncall}}", variables: map[string]string{"oncall": "  "}, wantErr: "empty value"},
		{name: "bad name", content: "{{oncall}}", variables: map[string]string{"On-Call": "x"}, wantErr: "lowercase letters"},
		{name: "value too long", content: "{{oncall}}", variables: map[string]string{"oncall": strings.Repeat("x", 501)}, wantErr: "limited to 500"},
		{name: "content too long", content: strings.Repeat("x", 10_001), wantErr: "limited to 10000"},
		{name: "trailing newline does not count", content: strings.Repeat("x", 10_000) + "\n"},
		{
			name:      "too long once filled in",
			content:   strings.Repeat("x", 9_990) + "{{v}}",
			variables: map[string]string{"v": strings.Repeat("y", 20)},
			wantErr:   "with its variables filled in",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diags := validateContent(tt.content, tt.variables, true)
			if tt.wantErr == "" {
				if diags.HasError() {
					t.Fatalf("unexpected error: %v", diags)
				}
				return
			}
			if !diags.HasError() {
				t.Fatalf("expected error containing %q", tt.wantErr)
			}
			if got := diags.Errors()[0].Detail(); !strings.Contains(got, tt.wantErr) {
				t.Errorf("error = %q, want it to contain %q", got, tt.wantErr)
			}
		})
	}
}

func TestValidateModel_CustomSlugNeedsTitleAndDescription(t *testing.T) {
	data := playbookModel{
		Slug:        types.StringValue("vendor-outages"),
		Title:       types.StringNull(),
		Description: types.StringNull(),
		Content:     types.StringValue("Check the vendor status page."),
		Variables:   types.MapNull(types.StringType),
	}
	diags := validateModel(context.Background(), data)
	if diags.ErrorsCount() != 2 {
		t.Fatalf("got %d errors, want 2 (title and description): %v", diags.ErrorsCount(), diags)
	}
}

func TestValidateModel_BuiltinSlugUsesDefaults(t *testing.T) {
	data := playbookModel{
		Slug:        types.StringValue("incident-response"),
		Title:       types.StringNull(),
		Description: types.StringNull(),
		Content:     types.StringValue("SEV1: page everyone."),
		Variables:   types.MapNull(types.StringType),
	}
	if diags := validateModel(context.Background(), data); diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
}

func TestValidateModel_UnknownSlugSkipsTitleCheck(t *testing.T) {
	data := playbookModel{
		Slug:        types.StringUnknown(),
		Title:       types.StringNull(),
		Description: types.StringNull(),
		Content:     types.StringValue("Anything."),
		Variables:   types.MapNull(types.StringType),
	}
	if diags := validateModel(context.Background(), data); diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
}

func TestValidateModel_UnknownVariableSkipsVariableChecks(t *testing.T) {
	m := types.MapValueMust(types.StringType, map[string]attr.Value{"unused": types.StringUnknown()})
	data := playbookModel{
		Slug:        types.StringValue("incident-response"),
		Title:       types.StringNull(),
		Description: types.StringNull(),
		Content:     types.StringValue("No references."),
		Variables:   m,
	}
	if diags := validateModel(context.Background(), data); diags.HasError() {
		t.Fatalf("unexpected error: %v", diags)
	}
}

func TestValidateModel_NullVariableIsReported(t *testing.T) {
	data := playbookModel{
		Slug:        types.StringValue("incident-response"),
		Title:       types.StringNull(),
		Description: types.StringNull(),
		Content:     types.StringValue("Page {{oncall}}."),
		Variables:   types.MapValueMust(types.StringType, map[string]attr.Value{"oncall": types.StringNull()}),
	}
	diags := validateModel(context.Background(), data)
	if !diags.HasError() {
		t.Fatal("expected an error for a null variable")
	}
	if got := diags.Errors()[0].Summary(); got != "Empty variable" {
		t.Errorf("summary = %q, want the provider's \"Empty variable\" error, not a conversion error", got)
	}
}

func TestNewUpdateRequest(t *testing.T) {
	data := playbookModel{
		Slug:        types.StringValue("status-page"),
		Title:       types.StringUnknown(),
		Description: types.StringNull(),
		Content:     types.StringValue("Be brief."),
	}
	req := newUpdateRequest(data, map[string]string{})
	if req.HasTitle() || req.HasDescription() {
		t.Error("unknown or null title/description must be left out so the API keeps the stored value")
	}
	if req.GetVariables() == nil {
		t.Error("variables must always be sent: UpdatePlaybook replaces them")
	}

	data.Title = types.StringValue("Status page voice")
	if req := newUpdateRequest(data, nil); !req.HasTitle() || req.GetTitle() != "Status page voice" {
		t.Error("known title must be sent")
	}
}

func apiPlaybook() *playbookv1.Playbook {
	p := &playbookv1.Playbook{}
	p.SetSlug("incident-response")
	p.SetTitle("Incident response")
	p.SetDescription("How we run incidents.")
	p.SetContent("Page {{oncall}}.")
	p.SetVariables(map[string]string{"oncall": "@sre"})
	p.SetCreatedAt("2026-01-01T00:00:00Z")
	p.SetUpdatedAt("2026-01-02T00:00:00Z")
	return p
}

// The API trims text; the configured value must survive so a heredoc's
// trailing newline doesn't make the apply inconsistent.
func TestPlaybookAPIToModel_KeepsUntrimmedConfig(t *testing.T) {
	vars := types.MapValueMust(types.StringType, map[string]attr.Value{"oncall": types.StringValue("@sre\n")})
	data := playbookModel{
		Title:       types.StringValue("  Incident response "),
		Description: types.StringUnknown(),
		Content:     types.StringValue("Page {{oncall}}.\n"),
		Variables:   vars,
	}
	if diags := playbookAPIToModel(context.Background(), apiPlaybook(), &data); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if data.Content.ValueString() != "Page {{oncall}}.\n" {
		t.Errorf("Content = %q, want the configured value", data.Content.ValueString())
	}
	if data.Title.ValueString() != "  Incident response " {
		t.Errorf("Title = %q, want the configured value", data.Title.ValueString())
	}
	if data.Description.ValueString() != "How we run incidents." {
		t.Errorf("Description = %q, want the API value", data.Description.ValueString())
	}
	if got := data.Variables.Elements()["oncall"].(types.String).ValueString(); got != "@sre\n" {
		t.Errorf("variables.oncall = %q, want the configured value", got)
	}
	if data.ID.ValueString() != "incident-response" || data.Slug.ValueString() != "incident-response" {
		t.Errorf("ID/Slug = %q/%q, want incident-response", data.ID.ValueString(), data.Slug.ValueString())
	}
}

// A real change made outside Terraform must show up as drift.
func TestPlaybookAPIToModel_ReportsRealDrift(t *testing.T) {
	data := playbookModel{
		Title:       types.StringValue("Old"),
		Description: types.StringValue("Old"),
		Content:     types.StringValue("Old content"),
		Variables:   types.MapNull(types.StringType),
	}
	if diags := playbookAPIToModel(context.Background(), apiPlaybook(), &data); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if data.Content.ValueString() != "Page {{oncall}}." {
		t.Errorf("Content = %q, want the API value", data.Content.ValueString())
	}
	if len(data.Variables.Elements()) != 1 {
		t.Errorf("Variables = %v, want the API value", data.Variables)
	}
}

func TestPlaybookAPIToModel_EmptyVariables(t *testing.T) {
	api := apiPlaybook()
	api.SetVariables(nil)

	data := playbookModel{Variables: types.MapNull(types.StringType)}
	playbookAPIToModel(context.Background(), api, &data)
	if !data.Variables.IsNull() {
		t.Errorf("Variables = %v, want null when omitted", data.Variables)
	}

	data = playbookModel{Variables: types.MapValueMust(types.StringType, map[string]attr.Value{})}
	playbookAPIToModel(context.Background(), api, &data)
	if data.Variables.IsNull() || len(data.Variables.Elements()) != 0 {
		t.Errorf("Variables = %v, want empty map when configured as {}", data.Variables)
	}
}
