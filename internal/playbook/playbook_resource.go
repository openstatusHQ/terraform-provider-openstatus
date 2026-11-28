package playbook

import (
	"context"
	"fmt"
	"strings"

	"terraform-provider-openstatus/internal/client"

	playbookv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/playbook/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                   = (*playbookResource)(nil)
	_ resource.ResourceWithConfigure      = (*playbookResource)(nil)
	_ resource.ResourceWithImportState    = (*playbookResource)(nil)
	_ resource.ResourceWithValidateConfig = (*playbookResource)(nil)
)

func NewPlaybookResource() resource.Resource {
	return &playbookResource{}
}

type playbookResource struct {
	client *client.Client
}

type playbookModel struct {
	ID          types.String `tfsdk:"id"`
	Slug        types.String `tfsdk:"slug"`
	Title       types.String `tfsdk:"title"`
	Description types.String `tfsdk:"description"`
	Content     types.String `tfsdk:"content"`
	Variables   types.Map    `tfsdk:"variables"`
	CreatedAt   types.String `tfsdk:"created_at"`
	UpdatedAt   types.String `tfsdk:"updated_at"`
}

func (r *playbookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_playbook"
}

func (r *playbookResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(providerConfig).Client
}

func builtinList() string {
	quoted := make([]string, len(BuiltinSlugs))
	for i, s := range BuiltinSlugs {
		quoted[i] = "`" + s + "`"
	}
	return strings.Join(quoted, ", ")
}

func (r *playbookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a playbook: markdown guidance the OpenStatus agent follows when it runs incidents, " +
			"writes status page updates, schedules maintenance or drafts postmortems.\n\n" +
			"The built-in slugs (" + builtinList() + ") come with a default title and description. " +
			"If the workspace has already written one, import it instead of creating it, so its content isn't overwritten. " +
			"Destroying a built-in makes the agent fall back to the OpenStatus default. Any other slug is a custom playbook and needs a `title` and `description`; " +
			"a workspace can have up to 20 custom playbooks.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Same as `slug`.",
				PlanModifiers:       []planmodifier.String{useStateIfSlugUnchanged()},
			},
			"slug": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "Identifies the playbook: lowercase letters, digits and single hyphens, max 64 characters. " +
					"Use a built-in slug (" + builtinList() + ") to override that playbook. Changing it forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators: []validator.String{
					stringvalidator.LengthAtMost(maxSlugChars),
					stringvalidator.RegexMatches(slugPattern, "must use lowercase letters, digits and single hyphens"),
				},
			},
			"title": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "Name of the playbook (1 to 100 characters). Required for custom playbooks. " +
					"For a built-in slug it defaults to the OpenStatus title when the playbook is first written. " +
					"Removing it from the configuration keeps the current value.",
				PlanModifiers: []planmodifier.String{useStateIfSlugUnchanged()},
			},
			"description": schema.StringAttribute{
				Optional: true,
				Computed: true,
				MarkdownDescription: "What the playbook covers (1 to 500 characters). The agent reads it to decide when to open the playbook. " +
					"Required for custom playbooks. For a built-in slug it defaults to the OpenStatus description when the playbook is first written. " +
					"Removing it from the configuration keeps the current value.",
				PlanModifiers: []planmodifier.String{useStateIfSlugUnchanged()},
			},
			"content": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The playbook in markdown, at most 10000 characters with its variables filled in. " +
					"`{{name}}` marks a variable. Leading and trailing whitespace is ignored.",
			},
			"variables": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				MarkdownDescription: "Values for the `{{name}}` variables in `content`, by name. Names use lowercase letters, digits and underscores; " +
					"values are at most 500 characters. Every variable must be referenced in `content` and have a non-empty value. " +
					"A variable referenced in `content` without a value is left as written.",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the playbook was first written (RFC 3339).",
				PlanModifiers:       []planmodifier.String{useStateIfSlugUnchanged()},
			},
			"updated_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the playbook was last changed (RFC 3339).",
			},
		},
	}
}

func (r *playbookResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data playbookModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(validateModel(ctx, data)...)
}

func validateModel(ctx context.Context, data playbookModel) diag.Diagnostics {
	var diags diag.Diagnostics

	if !data.Slug.IsNull() && !data.Slug.IsUnknown() && !IsBuiltinSlug(data.Slug.ValueString()) {
		for _, attr := range []struct {
			name  string
			value types.String
		}{{"title", data.Title}, {"description", data.Description}} {
			if attr.value.IsNull() {
				diags.AddAttributeError(path.Root(attr.name), "Missing "+attr.name,
					fmt.Sprintf("%q is a custom playbook, so %s is required. Only the built-in slugs (%s) have a default.",
						data.Slug.ValueString(), attr.name, strings.Join(BuiltinSlugs, ", ")))
			}
		}
	}

	if !data.Title.IsNull() && !data.Title.IsUnknown() {
		diags.Append(validateText(path.Root("title"), "Title", data.Title.ValueString(), maxTitleChars)...)
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		diags.Append(validateText(path.Root("description"), "Description", data.Description.ValueString(), maxDescriptionChars)...)
	}

	if !data.Content.IsNull() && !data.Content.IsUnknown() {
		variables, known, d := variablesFromModel(ctx, data.Variables)
		diags.Append(d...)
		if !d.HasError() {
			diags.Append(validateContent(data.Content.ValueString(), variables, known)...)
		}
	}

	return diags
}

// variablesFromModel reads the variables map. known is false while any value
// is still unknown at plan time.
func variablesFromModel(ctx context.Context, m types.Map) (map[string]string, bool, diag.Diagnostics) {
	out := map[string]string{}
	if m.IsNull() {
		return out, true, nil
	}
	if m.IsUnknown() {
		return out, false, nil
	}
	var diags diag.Diagnostics
	for name, v := range m.Elements() {
		if v.IsUnknown() {
			return out, false, nil
		}
		if v.IsNull() {
			diags.AddAttributeError(path.Root("variables").AtMapKey(name), "Empty variable",
				fmt.Sprintf("Variable %q is null. Remove it instead; OpenStatus does not store empty values.", name))
		}
	}
	if diags.HasError() {
		return out, true, diags
	}
	diags.Append(m.ElementsAs(ctx, &out, false)...)
	return out, true, diags
}

func (r *playbookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data, config playbookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	variables, _, diags := variablesFromModel(ctx, data.Variables)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var api *playbookv1.Playbook
	slug := data.Slug.ValueString()
	if IsBuiltinSlug(slug) {
		// CreatePlaybook refuses built-in slugs and UpdatePlaybook writes them,
		// but it would also overwrite a version the workspace already wrote
		// (and a later destroy would delete it). Refuse like a taken custom
		// slug does, so existing content is imported rather than lost.
		getReq := &playbookv1.GetPlaybookRequest{}
		getReq.SetSlug(slug)
		_, err := r.client.Playbook.GetPlaybook(ctx, connect.NewRequest(getReq))
		if err == nil {
			resp.Diagnostics.AddError("Playbook already exists",
				fmt.Sprintf("The workspace has already written the built-in %q playbook. To manage it without losing its content, import it: terraform import <address> %s", slug, slug))
			return
		}
		if !client.IsNotFound(err) {
			resp.Diagnostics.AddError("Error creating playbook", err.Error())
			return
		}

		apiResp, err := r.client.Playbook.UpdatePlaybook(ctx, connect.NewRequest(newUpdateRequest(config, variables)))
		if err != nil {
			resp.Diagnostics.AddError("Error creating playbook", err.Error())
			return
		}
		api = apiResp.Msg.GetPlaybook()
	} else {
		apiReq := &playbookv1.CreatePlaybookRequest{}
		apiReq.SetSlug(slug)
		apiReq.SetTitle(data.Title.ValueString())
		apiReq.SetDescription(data.Description.ValueString())
		apiReq.SetContent(data.Content.ValueString())
		apiReq.SetVariables(variables)

		apiResp, err := r.client.Playbook.CreatePlaybook(ctx, connect.NewRequest(apiReq))
		if err != nil {
			detail := err.Error()
			if connect.CodeOf(err) == connect.CodeAlreadyExists {
				detail += fmt.Sprintf("\n\nTo manage the existing playbook, import it: terraform import <address> %s", slug)
			}
			resp.Diagnostics.AddError("Error creating playbook", detail)
			return
		}
		api = apiResp.Msg.GetPlaybook()
	}

	resp.Diagnostics.Append(playbookAPIToModel(ctx, api, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *playbookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data playbookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &playbookv1.GetPlaybookRequest{}
	getReq.SetSlug(data.Slug.ValueString())

	apiResp, err := r.client.Playbook.GetPlaybook(ctx, connect.NewRequest(getReq))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading playbook", err.Error())
		return
	}

	resp.Diagnostics.Append(playbookAPIToModel(ctx, apiResp.Msg.GetPlaybook(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *playbookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, config playbookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	variables, _, diags := variablesFromModel(ctx, data.Variables)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := r.client.Playbook.UpdatePlaybook(ctx, connect.NewRequest(newUpdateRequest(config, variables)))
	if err != nil {
		resp.Diagnostics.AddError("Error updating playbook", err.Error())
		return
	}

	resp.Diagnostics.Append(playbookAPIToModel(ctx, apiResp.Msg.GetPlaybook(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *playbookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data playbookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &playbookv1.DeletePlaybookRequest{}
	deleteReq.SetSlug(data.Slug.ValueString())

	_, err := r.client.Playbook.DeletePlaybook(ctx, connect.NewRequest(deleteReq))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting playbook", err.Error())
	}
}

func (r *playbookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("slug"), types.StringValue(req.ID))...)
}

// newUpdateRequest builds the request from the configuration, not the plan.
// It always sends content and variables: UpdatePlaybook replaces both, so
// removing a variable from the configuration unsets it. Title and description
// are sent only when configured; left out, the API keeps the stored value (or
// the built-in default on a first write). The plan carries the stored value
// for an unset title, and sending it would pin a built-in's default.
func newUpdateRequest(data playbookModel, variables map[string]string) *playbookv1.UpdatePlaybookRequest {
	req := &playbookv1.UpdatePlaybookRequest{}
	req.SetSlug(data.Slug.ValueString())
	req.SetContent(data.Content.ValueString())
	req.SetVariables(variables)
	if !data.Title.IsNull() && !data.Title.IsUnknown() {
		req.SetTitle(data.Title.ValueString())
	}
	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		req.SetDescription(data.Description.ValueString())
	}
	return req
}

func playbookAPIToModel(ctx context.Context, api *playbookv1.Playbook, data *playbookModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(api.GetSlug())
	data.Slug = types.StringValue(api.GetSlug())
	data.Title = keepIfTrimEqual(data.Title, api.GetTitle())
	data.Description = keepIfTrimEqual(data.Description, api.GetDescription())
	data.Content = keepIfTrimEqual(data.Content, api.GetContent())
	data.CreatedAt = types.StringValue(api.GetCreatedAt())
	data.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	if len(api.GetVariables()) > 0 {
		prior, _, d := variablesFromModel(ctx, data.Variables)
		diags.Append(d...)
		m, d := types.MapValueFrom(ctx, types.StringType, keepVariablesIfTrimEqual(prior, api.GetVariables()))
		diags.Append(d...)
		data.Variables = m
	} else {
		data.Variables = emptyOrNullMap(ctx, data.Variables, &diags)
	}

	return diags
}

// emptyOrNullMap maps "no variables" back to the shape Terraform planned:
// null when the attribute was omitted, an empty map for `variables = {}`.
// variables is Optional and not Computed, so writing null over a planned empty
// map fails the apply with "Provider produced inconsistent result after apply".
func emptyOrNullMap(ctx context.Context, planned types.Map, diags *diag.Diagnostics) types.Map {
	if planned.IsNull() || planned.IsUnknown() {
		return types.MapNull(types.StringType)
	}
	m, d := types.MapValueFrom(ctx, types.StringType, map[string]string{})
	diags.Append(d...)
	return m
}

// useStateIfSlugUnchanged is UseStateForUnknown that stands down when the slug
// changes. A new slug is a new playbook, and carrying the old built-in's
// default title into the plan for another built-in would make the apply fail
// with an inconsistent result.
func useStateIfSlugUnchanged() planmodifier.String {
	return slugAwareUseState{}
}

type slugAwareUseState struct{}

func (slugAwareUseState) Description(context.Context) string {
	return "Uses the prior state value while the slug is unchanged."
}

func (m slugAwareUseState) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (slugAwareUseState) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	if req.StateValue.IsNull() || !req.PlanValue.IsUnknown() || req.ConfigValue.IsUnknown() {
		return
	}

	var planSlug, stateSlug types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("slug"), &planSlug)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("slug"), &stateSlug)...)
	if resp.Diagnostics.HasError() || planSlug.IsUnknown() || !planSlug.Equal(stateSlug) {
		return
	}

	resp.PlanValue = req.StateValue
}
