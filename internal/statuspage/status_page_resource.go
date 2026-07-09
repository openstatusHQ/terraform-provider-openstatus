package statuspage

import (
	"context"
	"fmt"
	"regexp"

	"terraform-provider-openstatus/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

var (
	_ resource.Resource                   = (*statusPageResource)(nil)
	_ resource.ResourceWithConfigure      = (*statusPageResource)(nil)
	_ resource.ResourceWithImportState    = (*statusPageResource)(nil)
	_ resource.ResourceWithValidateConfig = (*statusPageResource)(nil)
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
	AllowedIPRanges  types.String `tfsdk:"allowed_ip_ranges"`
	Published        types.Bool   `tfsdk:"published"`
	Theme            types.String `tfsdk:"theme"`
	CustomTheme      types.Object `tfsdk:"custom_theme"`
	DefaultLocale    types.String `tfsdk:"default_locale"`
	Locales          types.List   `tfsdk:"locales"`
	AllowIndex       types.Bool   `tfsdk:"allow_index"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

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
				Required: true,
				Validators: []validator.String{
					stringvalidator.LengthBetween(1, 256),
					stringvalidator.RegexMatches(
						slugPattern,
						`must be lowercase alphanumeric with hyphens between segments (e.g. "my-status-page")`,
					),
				},
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
				MarkdownDescription: "Access type of the status page. One of: `public`, `password`, `email-domain`, `ip`.",
				Validators:          []validator.String{stringvalidator.OneOf("public", "password", "email-domain", "ip")},
				PlanModifiers:       keepState,
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "Password to protect the status page. Required when `access_type` is `password`.",
			},
			"auth_email_domains": schema.ListAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "List of email domains allowed to access the page. Used when `access_type` is `email-domain`.",
			},
			"allowed_ip_ranges": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Comma-separated IPv4 CIDR ranges. Required when `access_type` is `ip`.",
			},
			"published": schema.BoolAttribute{Computed: true},
			"theme": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString("system"),
				Validators: []validator.String{stringvalidator.OneOf("system", "light", "dark")},
			},
			"custom_theme": schema.SingleNestedAttribute{
				Optional: true,
				MarkdownDescription: "Per-mode CSS variable overrides merged over `theme` (e.g. `--primary = \"hsl(24 94% 50%)\"`). " +
					"Only variable names supported by OpenStatus are accepted. Requires the custom-theme plan feature.",
				Attributes: map[string]schema.Attribute{
					"light": schema.MapAttribute{
						Optional:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "CSS variable overrides applied in light mode, keyed by variable name.",
						Validators:          []validator.Map{themeVarsValidator{}},
					},
					"dark": schema.MapAttribute{
						Optional:            true,
						ElementType:         types.StringType,
						MarkdownDescription: "CSS variable overrides applied in dark mode, keyed by variable name.",
						Validators:          []validator.Map{themeVarsValidator{}},
					},
				},
			},
			"default_locale": schema.StringAttribute{
				Optional:   true,
				Computed:   true,
				Default:    stringdefault.StaticString("en"),
				Validators: []validator.String{stringvalidator.OneOf("en", "fr", "de")},
			},
			"locales": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf("en", "fr", "de")),
				},
			},
			"allow_index": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *statusPageResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var data statusPageModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	accessType := data.AccessType.ValueString()
	hasPassword := !data.Password.IsNull() && !data.Password.IsUnknown()
	hasEmailDomains := !data.AuthEmailDomains.IsNull() && !data.AuthEmailDomains.IsUnknown()
	hasIPRanges := !data.AllowedIPRanges.IsNull() && !data.AllowedIPRanges.IsUnknown()

	if accessType == "password" && !hasPassword {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Missing required attribute",
			"password is required when access_type is \"password\".",
		)
	}
	if accessType == "email-domain" && !hasEmailDomains {
		resp.Diagnostics.AddAttributeError(
			path.Root("auth_email_domains"),
			"Missing required attribute",
			"auth_email_domains is required when access_type is \"email-domain\".",
		)
	}
	if accessType == "ip" && !hasIPRanges {
		resp.Diagnostics.AddAttributeError(
			path.Root("allowed_ip_ranges"),
			"Missing required attribute",
			"allowed_ip_ranges is required when access_type is \"ip\".",
		)
	}
	if hasPassword && accessType != "password" {
		resp.Diagnostics.AddAttributeError(
			path.Root("password"),
			"Invalid attribute combination",
			"password can only be set when access_type is \"password\".",
		)
	}
	if hasEmailDomains && accessType != "email-domain" {
		resp.Diagnostics.AddAttributeError(
			path.Root("auth_email_domains"),
			"Invalid attribute combination",
			"auth_email_domains can only be set when access_type is \"email-domain\".",
		)
	}
	if hasIPRanges && accessType != "ip" {
		resp.Diagnostics.AddAttributeError(
			path.Root("allowed_ip_ranges"),
			"Invalid attribute combination",
			"allowed_ip_ranges can only be set when access_type is \"ip\".",
		)
	}

	if !data.CustomTheme.IsNull() && !data.CustomTheme.IsUnknown() {
		var theme customThemeModel
		resp.Diagnostics.Append(data.CustomTheme.As(ctx, &theme, basetypes.ObjectAsOptions{})...)
		if resp.Diagnostics.HasError() {
			return
		}
		// The API stores an all-empty theme as "not configured", so state would never match.
		if theme.Light.IsNull() && theme.Dark.IsNull() {
			resp.Diagnostics.AddAttributeError(
				path.Root("custom_theme"),
				"Invalid custom theme",
				"custom_theme must set at least one of \"light\" or \"dark\"; remove the block instead of leaving it empty.",
			)
		}
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
	AllowedIPRanges  string   `json:"allowedIpRanges,omitempty"`
	DefaultLocale    string   `json:"defaultLocale,omitempty"`
	Locales          []string `json:"locales,omitempty"`
	AllowIndex       *bool    `json:"allowIndex,omitempty"`

	CustomTheme *apiCustomTheme `json:"customTheme,omitempty"`
}

type apiStatusPageUpdateRequest struct {
	ID               string    `json:"id"`
	Title            *string   `json:"title,omitempty"`
	Slug             *string   `json:"slug,omitempty"`
	Description      *string   `json:"description,omitempty"`
	HomepageURL      *string   `json:"homepageUrl,omitempty"`
	ContactURL       *string   `json:"contactUrl,omitempty"`
	Icon             *string   `json:"icon,omitempty"`
	CustomDomain     *string   `json:"customDomain,omitempty"`
	AccessType       *string   `json:"accessType,omitempty"`
	Password         *string   `json:"password,omitempty"`
	AuthEmailDomains *[]string `json:"authEmailDomains,omitempty"`
	AllowedIPRanges  *string   `json:"allowedIpRanges,omitempty"`
	Theme            *string   `json:"theme,omitempty"`
	DefaultLocale    *string   `json:"defaultLocale,omitempty"`
	Locales          *[]string `json:"locales,omitempty"`
	AllowIndex       *bool     `json:"allowIndex,omitempty"`

	// Omitted = keep current value; empty message = clear.
	CustomTheme *apiCustomTheme `json:"customTheme,omitempty"`
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
	AllowedIPRanges  string   `json:"allowedIpRanges"`
	Theme            string   `json:"theme"`
	DefaultLocale    string   `json:"defaultLocale"`
	Locales          []string `json:"locales"`
	AllowIndex       bool     `json:"allowIndex"`
	CreatedAt        string   `json:"createdAt"`
	UpdatedAt        string   `json:"updatedAt"`

	CustomTheme *apiCustomTheme `json:"customTheme"`
}

func (r *statusPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data statusPageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := apiStatusPageCreateRequest{
		Title:           data.Title.ValueString(),
		Slug:            data.Slug.ValueString(),
		Description:     data.Description.ValueString(),
		HomepageURL:     data.HomepageURL.ValueString(),
		ContactURL:      data.ContactURL.ValueString(),
		Icon:            data.Icon.ValueString(),
		CustomDomain:    data.CustomDomain.ValueString(),
		AccessType:      accessTypeToProto(data.AccessType.ValueString()),
		Password:        data.Password.ValueString(),
		AllowedIPRanges: data.AllowedIPRanges.ValueString(),
		Theme:           themeToProto(data.Theme.ValueString()),
		DefaultLocale:   localeToProto(data.DefaultLocale.ValueString()),
	}

	if !data.AuthEmailDomains.IsNull() && !data.AuthEmailDomains.IsUnknown() {
		var domains []string
		resp.Diagnostics.Append(data.AuthEmailDomains.ElementsAs(ctx, &domains, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiReq.AuthEmailDomains = domains
	}

	if !data.Locales.IsNull() && !data.Locales.IsUnknown() {
		var tfLocales []string
		resp.Diagnostics.Append(data.Locales.ElementsAs(ctx, &tfLocales, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiLocales := make([]string, 0, len(tfLocales))
		for _, l := range tfLocales {
			apiLocales = append(apiLocales, localeToProto(l))
		}
		apiReq.Locales = apiLocales
	}

	if !data.AllowIndex.IsNull() && !data.AllowIndex.IsUnknown() {
		v := data.AllowIndex.ValueBool()
		apiReq.AllowIndex = &v
	}

	apiReq.CustomTheme = customThemeToAPI(ctx, data.CustomTheme, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
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
	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		v := data.Password.ValueString()
		updateReq.Password = &v
	}

	if !data.AccessType.IsNull() && !data.AccessType.IsUnknown() {
		v := accessTypeToProto(data.AccessType.ValueString())
		updateReq.AccessType = &v
	}
	if !data.AuthEmailDomains.IsNull() && !data.AuthEmailDomains.IsUnknown() {
		var domains []string
		resp.Diagnostics.Append(data.AuthEmailDomains.ElementsAs(ctx, &domains, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.AuthEmailDomains = &domains
	}
	if !data.AllowedIPRanges.IsNull() && !data.AllowedIPRanges.IsUnknown() {
		v := data.AllowedIPRanges.ValueString()
		updateReq.AllowedIPRanges = &v
	}
	if !data.Theme.IsNull() && !data.Theme.IsUnknown() {
		v := themeToProto(data.Theme.ValueString())
		updateReq.Theme = &v
	}
	if !data.DefaultLocale.IsNull() && !data.DefaultLocale.IsUnknown() {
		v := localeToProto(data.DefaultLocale.ValueString())
		updateReq.DefaultLocale = &v
	}
	if !data.Locales.IsNull() && !data.Locales.IsUnknown() {
		var tfLocales []string
		resp.Diagnostics.Append(data.Locales.ElementsAs(ctx, &tfLocales, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiLocales := make([]string, 0, len(tfLocales))
		for _, l := range tfLocales {
			apiLocales = append(apiLocales, localeToProto(l))
		}
		updateReq.Locales = &apiLocales
	}
	if !data.AllowIndex.IsNull() && !data.AllowIndex.IsUnknown() {
		v := data.AllowIndex.ValueBool()
		updateReq.AllowIndex = &v
	}

	if !data.CustomTheme.IsNull() && !data.CustomTheme.IsUnknown() {
		updateReq.CustomTheme = customThemeToAPI(ctx, data.CustomTheme, &resp.Diagnostics)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		// Clear only when state had one: the field's presence triggers the
		// server's plan-feature check, breaking unrelated updates otherwise.
		var state statusPageModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !state.CustomTheme.IsNull() {
			updateReq.CustomTheme = &apiCustomTheme{}
		}
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

func accessTypeToProto(tf string) string {
	switch tf {
	case "public":
		return "PAGE_ACCESS_TYPE_PUBLIC"
	case "password":
		return "PAGE_ACCESS_TYPE_PASSWORD_PROTECTED"
	case "email-domain":
		return "PAGE_ACCESS_TYPE_AUTHENTICATED"
	case "ip":
		return "PAGE_ACCESS_TYPE_IP_RESTRICTED"
	}
	return ""
}

func accessTypeFromProto(proto string) (string, bool) {
	switch proto {
	case "PAGE_ACCESS_TYPE_PUBLIC":
		return "public", true
	case "PAGE_ACCESS_TYPE_PASSWORD_PROTECTED":
		return "password", true
	case "PAGE_ACCESS_TYPE_AUTHENTICATED":
		return "email-domain", true
	case "PAGE_ACCESS_TYPE_IP_RESTRICTED":
		return "ip", true
	}
	return "", false
}

func themeToProto(tf string) string {
	switch tf {
	case "system":
		return "PAGE_THEME_SYSTEM"
	case "light":
		return "PAGE_THEME_LIGHT"
	case "dark":
		return "PAGE_THEME_DARK"
	}
	return ""
}

func themeFromProto(proto string) (string, bool) {
	switch proto {
	case "PAGE_THEME_SYSTEM":
		return "system", true
	case "PAGE_THEME_LIGHT":
		return "light", true
	case "PAGE_THEME_DARK":
		return "dark", true
	}
	return "", false
}

func localeToProto(tf string) string {
	switch tf {
	case "en":
		return "LOCALE_EN"
	case "fr":
		return "LOCALE_FR"
	case "de":
		return "LOCALE_DE"
	}
	return ""
}

func localeFromProto(proto string) (string, bool) {
	switch proto {
	case "LOCALE_EN":
		return "en", true
	case "LOCALE_FR":
		return "fr", true
	case "LOCALE_DE":
		return "de", true
	}
	return "", false
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
	if v, ok := accessTypeFromProto(api.AccessType); ok {
		data.AccessType = types.StringValue(v)
	} else if api.AccessType != "" {
		diags.AddWarning(
			"Unknown status page access type",
			fmt.Sprintf("OpenStatus returned access_type %q which this provider version does not recognize.", api.AccessType),
		)
	}
	if api.Password != "" {
		data.Password = types.StringValue(api.Password)
	}
	if len(api.AuthEmailDomains) > 0 || !data.AuthEmailDomains.IsNull() {
		list, listDiags := types.ListValueFrom(ctx, types.StringType, api.AuthEmailDomains)
		diags.Append(listDiags...)
		data.AuthEmailDomains = list
	}
	if v, ok := themeFromProto(api.Theme); ok {
		data.Theme = types.StringValue(v)
	} else if api.Theme != "" {
		diags.AddWarning(
			"Unknown status page theme",
			fmt.Sprintf("OpenStatus returned theme %q which this provider version does not recognize.", api.Theme),
		)
	}
	data.CustomTheme = customThemeFromAPI(ctx, api.CustomTheme, diags)
	if v, ok := localeFromProto(api.DefaultLocale); ok {
		data.DefaultLocale = types.StringValue(v)
	} else if api.DefaultLocale != "" {
		diags.AddWarning(
			"Unknown status page default locale",
			fmt.Sprintf("OpenStatus returned default_locale %q which this provider version does not recognize.", api.DefaultLocale),
		)
	}
	if len(api.Locales) > 0 || !data.Locales.IsNull() {
		tfLocales := make([]string, 0, len(api.Locales))
		for _, l := range api.Locales {
			if v, ok := localeFromProto(l); ok {
				tfLocales = append(tfLocales, v)
			} else {
				diags.AddWarning(
					"Unknown status page locale",
					fmt.Sprintf("OpenStatus returned locale %q which this provider version does not recognize.", l),
				)
			}
		}
		list, listDiags := types.ListValueFrom(ctx, types.StringType, tfLocales)
		diags.Append(listDiags...)
		data.Locales = list
	}
	data.AllowIndex = types.BoolValue(api.AllowIndex)
	if api.AllowedIPRanges != "" || !data.AllowedIPRanges.IsNull() {
		data.AllowedIPRanges = types.StringValue(api.AllowedIPRanges)
	}
	data.CreatedAt = types.StringValue(api.CreatedAt)
	data.UpdatedAt = types.StringValue(api.UpdatedAt)
}

func isNotFound(err error) bool {
	if apiErr, ok := err.(*client.APIError); ok {
		return apiErr.Code == "not_found"
	}
	return false
}
