package statuspage

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*statusPageResource)(nil)
	_ resource.ResourceWithConfigure   = (*statusPageResource)(nil)
	_ resource.ResourceWithImportState = (*statusPageResource)(nil)
)

type providerConfig = client.ProviderConfig

func NewStatusPageResource() resource.Resource {
	return &statusPageResource{}
}

type statusPageResource struct {
	client *client.Client
}

type statusPageModel struct {
	ID           types.String `tfsdk:"id"`
	Title        types.String `tfsdk:"title"`
	Slug         types.String `tfsdk:"slug"`
	Description  types.String `tfsdk:"description"`
	HomepageURL  types.String `tfsdk:"homepage_url"`
	ContactURL   types.String `tfsdk:"contact_url"`
	CustomDomain types.String `tfsdk:"custom_domain"`
	Published    types.Bool   `tfsdk:"published"`
	AccessType   types.String `tfsdk:"access_type"`
	Theme        types.String `tfsdk:"theme"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (r *statusPageResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (r *statusPageResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(providerConfig).Client
}

func (r *statusPageResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	requiresReplace := []planmodifier.String{stringplanmodifier.RequiresReplace()}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a status page. Note: the API does not support updates, so any change to mutable attributes will destroy and recreate the resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"title": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.LengthBetween(1, 256)},
				PlanModifiers: requiresReplace,
			},
			"slug": schema.StringAttribute{
				Required:      true,
				Validators:    []validator.String{stringvalidator.LengthBetween(1, 256)},
				PlanModifiers: requiresReplace,
			},
			"description": schema.StringAttribute{
				Optional:      true,
				Validators:    []validator.String{stringvalidator.LengthAtMost(1024)},
				PlanModifiers: requiresReplace,
			},
			"homepage_url": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: requiresReplace,
			},
			"contact_url": schema.StringAttribute{
				Optional:      true,
				PlanModifiers: requiresReplace,
			},
			"custom_domain": schema.StringAttribute{Computed: true},
			"published":     schema.BoolAttribute{Computed: true},
			"access_type":   schema.StringAttribute{Computed: true},
			"theme":         schema.StringAttribute{Computed: true},
			"created_at":    schema.StringAttribute{Computed: true},
			"updated_at":    schema.StringAttribute{Computed: true},
		},
	}
}

type apiStatusPageCreateRequest struct {
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	HomepageURL string `json:"homepageUrl,omitempty"`
	ContactURL  string `json:"contactUrl,omitempty"`
}

type apiStatusPageResponse struct {
	StatusPage apiStatusPage `json:"statusPage"`
}

type apiStatusPage struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Slug         string `json:"slug"`
	Description  string `json:"description"`
	HomepageURL  string `json:"homepageUrl"`
	ContactURL   string `json:"contactUrl"`
	CustomDomain string `json:"customDomain"`
	Published    bool   `json:"published"`
	AccessType   string `json:"accessType"`
	Theme        string `json:"theme"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

func (r *statusPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data statusPageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := apiStatusPageCreateRequest{
		Title:       data.Title.ValueString(),
		Slug:        data.Slug.ValueString(),
		Description: data.Description.ValueString(),
		HomepageURL: data.HomepageURL.ValueString(),
		ContactURL:  data.ContactURL.ValueString(),
	}

	var apiResp apiStatusPageResponse
	err := r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/CreateStatusPage", apiReq, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error creating status page", err.Error())
		return
	}

	statusPageAPIToModel(apiResp.StatusPage, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *statusPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data statusPageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResp apiStatusPageResponse
	err := r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/GetStatusPage",
		map[string]string{"id": data.ID.ValueString()}, &apiResp)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading status page", err.Error())
		return
	}

	statusPageAPIToModel(apiResp.StatusPage, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *statusPageResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All mutable fields have RequiresReplace, so Update is never called.
}

func (r *statusPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data statusPageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/DeleteStatusPage",
		map[string]string{"id": data.ID.ValueString()}, nil)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting status page", err.Error())
	}
}

func (r *statusPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
}

func statusPageAPIToModel(api apiStatusPage, data *statusPageModel) {
	data.ID = types.StringValue(api.ID)
	data.Title = types.StringValue(api.Title)
	data.Slug = types.StringValue(api.Slug)
	if api.Description != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(api.Description)
	}
	if api.HomepageURL != "" || !data.HomepageURL.IsNull() {
		data.HomepageURL = types.StringValue(api.HomepageURL)
	}
	if api.ContactURL != "" || !data.ContactURL.IsNull() {
		data.ContactURL = types.StringValue(api.ContactURL)
	}
	data.CustomDomain = types.StringValue(api.CustomDomain)
	data.Published = types.BoolValue(api.Published)
	data.AccessType = types.StringValue(api.AccessType)
	data.Theme = types.StringValue(api.Theme)
	data.CreatedAt = types.StringValue(api.CreatedAt)
	data.UpdatedAt = types.StringValue(api.UpdatedAt)
}

func isNotFound(err error) bool {
	if apiErr, ok := err.(*client.APIError); ok {
		return apiErr.Code == "not_found"
	}
	return false
}

