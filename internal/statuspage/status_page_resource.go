package statuspage

import (
	"context"
	"fmt"
	"net/http"

	"terraform-provider-openstatus/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
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
	ID               types.String `tfsdk:"id"`
	Title            types.String `tfsdk:"title"`
	Slug             types.String `tfsdk:"slug"`
	Description      types.String `tfsdk:"description"`
	HomepageURL      types.String `tfsdk:"homepage_url"`
	ContactURL       types.String `tfsdk:"contact_url"`
	Icon             types.String `tfsdk:"icon"`
	CustomDomain     types.String `tfsdk:"custom_domain"`
	AccessType       types.String `tfsdk:"access_type"`
	Password         types.String `tfsdk:"password"`
	AuthEmailDomains types.List   `tfsdk:"auth_email_domains"`
	Published        types.Bool   `tfsdk:"published"`
	Theme            types.String `tfsdk:"theme"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
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
		MarkdownDescription: "Manages a status page. Any change to mutable attributes will destroy and recreate the resource.",
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
			"icon": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "URL of the icon to display on the status page.",
				PlanModifiers:       requiresReplace,
			},
			"custom_domain": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Custom domain for the status page. Must be configured within the dashboard.",
				PlanModifiers:       requiresReplace,
			},
			"access_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Access type of the status page. One of: `public`, `password`, `email-domain`.",
				Validators:          []validator.String{stringvalidator.OneOf("public", "password", "email-domain")},
				PlanModifiers:       requiresReplace,
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Password to protect the status page. Required when `access_type` is `password`.",
				PlanModifiers:       requiresReplace,
			},
			"auth_email_domains": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of email domains allowed to access the page. Used when `access_type` is `email-domain`.",
				PlanModifiers:       []planmodifier.List{listplanmodifier.RequiresReplace()},
			},
			"published": schema.BoolAttribute{Computed: true},
			"theme":     schema.StringAttribute{Computed: true},
			"created_at":    schema.StringAttribute{Computed: true},
			"updated_at":    schema.StringAttribute{Computed: true},
		},
	}
}

// apiRPCCreateRequest is sent to the RPC CreateStatusPage endpoint.
type apiRPCCreateRequest struct {
	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	HomepageURL string `json:"homepageUrl,omitempty"`
	ContactURL  string `json:"contactUrl,omitempty"`
}

// apiRPCResponse wraps the RPC response which nests the page under "statusPage".
type apiRPCResponse struct {
	StatusPage apiRPCStatusPage `json:"statusPage"`
}

// apiRPCStatusPage is the page object returned by the RPC API (string ID).
type apiRPCStatusPage struct {
	ID string `json:"id"`
}

// apiRESTUpdateRequest is sent to the REST v1 PUT endpoint for fields not supported by RPC.
type apiRESTUpdateRequest struct {
	Icon             string   `json:"icon,omitempty"`
	CustomDomain     string   `json:"customDomain,omitempty"`
	AccessType       string   `json:"accessType,omitempty"`
	Password         string   `json:"password,omitempty"`
	AuthEmailDomains []string `json:"authEmailDomains,omitempty"`
}

// apiRESTStatusPage is the flat JSON response from the REST v1 API (numeric ID).
type apiRESTStatusPage struct {
	ID               int      `json:"id"`
	Title            string   `json:"title"`
	Slug             string   `json:"slug"`
	Description      string   `json:"description"`
	Icon             string   `json:"icon"`
	CustomDomain     string   `json:"customDomain"`
	Published        bool     `json:"published"`
	AccessType       string   `json:"accessType"`
	Password         string   `json:"password"`
	AuthEmailDomains []string `json:"authEmailDomains"`
	Theme            string   `json:"theme"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`
}

func (r *statusPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data statusPageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Step 1: Create via RPC (supports homepageUrl/contactUrl).
	rpcReq := apiRPCCreateRequest{
		Title:       data.Title.ValueString(),
		Slug:        data.Slug.ValueString(),
		Description: data.Description.ValueString(),
		HomepageURL: data.HomepageURL.ValueString(),
		ContactURL:  data.ContactURL.ValueString(),
	}

	var rpcResp apiRPCResponse
	err := r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/CreateStatusPage", rpcReq, &rpcResp)
	if err != nil {
		resp.Diagnostics.AddError("Error creating status page", err.Error())
		return
	}

	pageID := rpcResp.StatusPage.ID
	// Save ID immediately so Terraform can track/delete the resource on retry.
	data.ID = types.StringValue(pageID)

	// Step 2: If REST-only fields are set, update via REST PUT.
	restReq := buildRESTUpdateRequest(&data, ctx, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if restReq != nil {
		err = r.client.DoREST(ctx, http.MethodPut, fmt.Sprintf("/page/%s", pageID), restReq, nil)
		if err != nil {
			// Save partial state so the resource can be cleaned up.
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
			resp.Diagnostics.AddError("Error updating status page (REST)", err.Error())
			return
		}
	}

	// Step 3: Read back full state via REST.
	var apiResp apiRESTStatusPage
	err = r.client.DoREST(ctx, http.MethodGet, fmt.Sprintf("/page/%s", pageID), nil, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error reading status page after create", err.Error())
		return
	}

	restAPIToModel(ctx, apiResp, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *statusPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data statusPageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResp apiRESTStatusPage
	err := r.client.DoREST(ctx, http.MethodGet, fmt.Sprintf("/page/%s", data.ID.ValueString()), nil, &apiResp)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading status page", err.Error())
		return
	}

	restAPIToModel(ctx, apiResp, &data, &resp.Diagnostics)
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

// buildRESTUpdateRequest returns a REST update request for fields not supported by the RPC API.
// Returns nil if no REST-only fields are set.
func buildRESTUpdateRequest(data *statusPageModel, ctx context.Context, diags *diag.Diagnostics) *apiRESTUpdateRequest {
	req := &apiRESTUpdateRequest{
		Icon:         data.Icon.ValueString(),
		CustomDomain: data.CustomDomain.ValueString(),
		AccessType:   data.AccessType.ValueString(),
		Password:     data.Password.ValueString(),
	}

	if !data.AuthEmailDomains.IsNull() && !data.AuthEmailDomains.IsUnknown() {
		var domains []string
		diags.Append(data.AuthEmailDomains.ElementsAs(ctx, &domains, false)...)
		req.AuthEmailDomains = domains
	}

	// Only return the request if at least one field is set.
	if req.Icon == "" && req.CustomDomain == "" && req.AccessType == "" && req.Password == "" && req.AuthEmailDomains == nil {
		return nil
	}
	return req
}

// restAPIToModel maps a REST v1 API response to the Terraform model.
// Fields not returned by the REST API (homepageUrl, contactUrl) are preserved from the existing model.
func restAPIToModel(ctx context.Context, api apiRESTStatusPage, data *statusPageModel, diags *diag.Diagnostics) {
	data.ID = types.StringValue(fmt.Sprintf("%d", api.ID))
	data.Title = types.StringValue(api.Title)
	data.Slug = types.StringValue(api.Slug)
	if api.Description != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(api.Description)
	}
	// homepageUrl and contactUrl are not returned by the REST API — preserve from state.
	if api.Icon != "" || !data.Icon.IsNull() {
		data.Icon = types.StringValue(api.Icon)
	}
	if api.CustomDomain != "" || !data.CustomDomain.IsNull() {
		data.CustomDomain = types.StringValue(api.CustomDomain)
	}
	data.Published = types.BoolValue(api.Published)
	data.AccessType = types.StringValue(api.AccessType)
	if api.Password != "" || !data.Password.IsNull() {
		data.Password = types.StringValue(api.Password)
	}
	if len(api.AuthEmailDomains) > 0 || !data.AuthEmailDomains.IsNull() {
		list, listDiags := types.ListValueFrom(ctx, types.StringType, api.AuthEmailDomains)
		diags.Append(listDiags...)
		data.AuthEmailDomains = list
	}
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

