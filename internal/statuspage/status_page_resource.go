package statuspage

import (
	"context"

	"terraform-provider-openstatus/internal/client"

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
	keepState := []planmodifier.String{stringplanmodifier.UseStateForUnknown()}

	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a status page.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: keepState,
			},
			"title": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, 256)},
			},
			"slug": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, 256)},
			},
			"description": schema.StringAttribute{
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthAtMost(1024)},
			},
			"homepage_url": schema.StringAttribute{
				Optional: true,
			},
			"contact_url": schema.StringAttribute{
				Optional: true,
			},
			"icon": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "URL of the icon to display on the status page.",
			},
			"custom_domain": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Custom domain for the status page. DNS must point to OpenStatus before setting this.",
			},
			"access_type": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Access type of the status page. One of: `public`, `password`, `email-domain`.",
				Validators:          []validator.String{stringvalidator.OneOf("public", "password", "email-domain")},
				PlanModifiers:       keepState,
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Password to protect the status page. Required when `access_type` is `password`.",
			},
			"auth_email_domains": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				MarkdownDescription: "List of email domains allowed to access the page. Used when `access_type` is `email-domain`.",
			},
			"published": schema.BoolAttribute{Computed: true},
			"theme":     schema.StringAttribute{Computed: true},
			"created_at":    schema.StringAttribute{Computed: true},
			"updated_at":    schema.StringAttribute{Computed: true},
		},
	}
}

type apiStatusPageCreateRequest struct {
	Title            string   `json:"title"`
	Slug             string   `json:"slug"`
	Description      string   `json:"description,omitempty"`
	HomepageURL      string   `json:"homepageUrl,omitempty"`
	ContactURL       string   `json:"contactUrl,omitempty"`
	Icon             string   `json:"icon,omitempty"`
	CustomDomain     string   `json:"customDomain,omitempty"`
	Theme            string   `json:"theme,omitempty"`
	AccessType       string   `json:"accessType,omitempty"`
	Password         string   `json:"password,omitempty"`
	AuthEmailDomains []string `json:"authEmailDomains,omitempty"`
}

type apiStatusPageUpdateRequest struct {
	ID               string   `json:"id"`
	Title            *string  `json:"title,omitempty"`
	Slug             *string  `json:"slug,omitempty"`
	Description      *string  `json:"description,omitempty"`
	HomepageURL      *string  `json:"homepageUrl,omitempty"`
	ContactURL       *string  `json:"contactUrl,omitempty"`
	Icon             *string  `json:"icon,omitempty"`
	CustomDomain     *string  `json:"customDomain,omitempty"`
	AccessType       *string  `json:"accessType,omitempty"`
	Password         *string  `json:"password,omitempty"`
	AuthEmailDomains []string `json:"authEmailDomains,omitempty"`
}

type apiStatusPageResponse struct {
	StatusPage apiStatusPage `json:"statusPage"`
}

type apiStatusPage struct {
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Slug             string   `json:"slug"`
	Description      string   `json:"description"`
	HomepageURL      string   `json:"homepageUrl"`
	ContactURL       string   `json:"contactUrl"`
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

	apiReq := apiStatusPageCreateRequest{
		Title:       data.Title.ValueString(),
		Slug:        data.Slug.ValueString(),
		Description: data.Description.ValueString(),
		HomepageURL: data.HomepageURL.ValueString(),
		ContactURL:  data.ContactURL.ValueString(),
		Icon:        data.Icon.ValueString(),
		CustomDomain: data.CustomDomain.ValueString(),
		AccessType:  accessTypeToProto(data.AccessType.ValueString()),
		Password:    data.Password.ValueString(),
	}

	if !data.AuthEmailDomains.IsNull() && !data.AuthEmailDomains.IsUnknown() {
		var domains []string
		resp.Diagnostics.Append(data.AuthEmailDomains.ElementsAs(ctx, &domains, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiReq.AuthEmailDomains = domains
	}

	var apiResp apiStatusPageResponse
	err := r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/CreateStatusPage", apiReq, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error creating status page", err.Error())
		return
	}

	pageID := apiResp.StatusPage.ID
	data.ID = types.StringValue(pageID)

	// Read back full state.
	err = r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/GetStatusPage",
		map[string]string{"id": pageID}, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error reading status page after create", err.Error())
		return
	}

	statusPageAPIToModel(ctx, apiResp.StatusPage, &data, &resp.Diagnostics)
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

	statusPageAPIToModel(ctx, apiResp.StatusPage, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *statusPageResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data statusPageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Send all fields on every update. Null in plan means "clear the field" (empty string).
	title := data.Title.ValueString()
	slug := data.Slug.ValueString()
	desc := data.Description.ValueString()
	homepageURL := data.HomepageURL.ValueString()
	contactURL := data.ContactURL.ValueString()
	icon := data.Icon.ValueString()
	customDomain := data.CustomDomain.ValueString()

	updateReq := apiStatusPageUpdateRequest{
		ID:           data.ID.ValueString(),
		Title:        &title,
		Slug:         &slug,
		Description:  &desc,
		HomepageURL:  &homepageURL,
		ContactURL:   &contactURL,
		Icon:         &icon,
		CustomDomain: &customDomain,
	}

	// Password has a min_len=1 validation server-side, only send when set.
	if !data.Password.IsNull() {
		v := data.Password.ValueString()
		updateReq.Password = &v
	}

	if !data.AccessType.IsNull() && !data.AccessType.IsUnknown() {
		v := accessTypeToProto(data.AccessType.ValueString())
		updateReq.AccessType = &v
	}
	if !data.AuthEmailDomains.IsNull() {
		var domains []string
		resp.Diagnostics.Append(data.AuthEmailDomains.ElementsAs(ctx, &domains, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.AuthEmailDomains = domains
	}

	var apiResp apiStatusPageResponse
	err := r.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/UpdateStatusPage", updateReq, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error updating status page", err.Error())
		return
	}

	statusPageAPIToModel(ctx, apiResp.StatusPage, &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
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

// accessTypeToProto converts a Terraform access_type value to the proto enum string.
func accessTypeToProto(tf string) string {
	switch tf {
	case "public":
		return "PAGE_ACCESS_TYPE_PUBLIC"
	case "password":
		return "PAGE_ACCESS_TYPE_PASSWORD_PROTECTED"
	case "email-domain":
		return "PAGE_ACCESS_TYPE_AUTHENTICATED"
	default:
		return ""
	}
}

// accessTypeFromProto converts a proto enum string to the Terraform access_type value.
func accessTypeFromProto(proto string) string {
	switch proto {
	case "PAGE_ACCESS_TYPE_PUBLIC":
		return "public"
	case "PAGE_ACCESS_TYPE_PASSWORD_PROTECTED":
		return "password"
	case "PAGE_ACCESS_TYPE_AUTHENTICATED":
		return "email-domain"
	default:
		return "public"
	}
}

// themeFromProto converts a proto theme enum string to a friendly Terraform value.
func themeFromProto(proto string) string {
	switch proto {
	case "PAGE_THEME_SYSTEM":
		return "system"
	case "PAGE_THEME_LIGHT":
		return "light"
	case "PAGE_THEME_DARK":
		return "dark"
	default:
		return "system"
	}
}

// statusPageAPIToModel maps an API response to the Terraform model.
func statusPageAPIToModel(ctx context.Context, api apiStatusPage, data *statusPageModel, diags *diag.Diagnostics) {
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
	if api.Icon != "" || !data.Icon.IsNull() {
		data.Icon = types.StringValue(api.Icon)
	}
	if api.CustomDomain != "" || !data.CustomDomain.IsNull() {
		data.CustomDomain = types.StringValue(api.CustomDomain)
	}
	data.Published = types.BoolValue(api.Published)
	data.AccessType = types.StringValue(accessTypeFromProto(api.AccessType))
	if api.Password != "" {
		data.Password = types.StringValue(api.Password)
	}
	if len(api.AuthEmailDomains) > 0 || !data.AuthEmailDomains.IsNull() {
		list, listDiags := types.ListValueFrom(ctx, types.StringType, api.AuthEmailDomains)
		diags.Append(listDiags...)
		data.AuthEmailDomains = list
	}
	data.Theme = types.StringValue(themeFromProto(api.Theme))
	data.CreatedAt = types.StringValue(api.CreatedAt)
	data.UpdatedAt = types.StringValue(api.UpdatedAt)
}

func isNotFound(err error) bool {
	if apiErr, ok := err.(*client.APIError); ok {
		return apiErr.Code == "not_found"
	}
	return false
}
