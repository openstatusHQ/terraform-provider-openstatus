package statuspage

import (
	"context"
	"fmt"
	"regexp"

	"terraform-provider-openstatus/internal/client"

	statuspagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"

	"connectrpc.com/connect"

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
				MarkdownDescription: "Per-mode CSS variable overrides merged over `theme` (e.g. `\"--primary\" = \"hsl(24 94% 50%)\"`). " +
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
				Validators: []validator.String{stringvalidator.OneOf("en", "fr", "de", "hi", "ko", "tr")},
			},
			"locales": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				MarkdownDescription: "Locales the status page is available in, in display order. One of: `en`, `fr`, `de`. " +
					"Each locale may only be listed once; the API stores locales as a set and would silently drop duplicates.",
				Validators: []validator.List{
					listvalidator.ValueStringsAre(stringvalidator.OneOf("en", "fr", "de", "hi", "ko", "tr")),
					listvalidator.UniqueValues(),
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

func (r *statusPageResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data statusPageModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Optional fields stay absent when empty. Sending an explicit zero here
	// would tell the server "set this to unspecified" rather than "use the
	// default", which for access_type and theme is a different outcome.
	apiReq := &statuspagev1.CreateStatusPageRequest{}
	apiReq.SetTitle(data.Title.ValueString())
	apiReq.SetSlug(data.Slug.ValueString())
	if v := data.Description.ValueString(); v != "" {
		apiReq.SetDescription(v)
	}
	if v := data.HomepageURL.ValueString(); v != "" {
		apiReq.SetHomepageUrl(v)
	}
	if v := data.ContactURL.ValueString(); v != "" {
		apiReq.SetContactUrl(v)
	}
	if v := data.Icon.ValueString(); v != "" {
		apiReq.SetIcon(v)
	}
	if v := data.CustomDomain.ValueString(); v != "" {
		apiReq.SetCustomDomain(v)
	}
	if v := accessTypeToProto(data.AccessType.ValueString()); v != statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_UNSPECIFIED {
		apiReq.SetAccessType(v)
	}
	if v := data.Password.ValueString(); v != "" {
		apiReq.SetPassword(v)
	}
	if v := data.AllowedIPRanges.ValueString(); v != "" {
		apiReq.SetAllowedIpRanges(v)
	}
	if v := themeToProto(data.Theme.ValueString()); v != statuspagev1.PageTheme_PAGE_THEME_UNSPECIFIED {
		apiReq.SetTheme(v)
	}
	if v := localeToProto(data.DefaultLocale.ValueString()); v != statuspagev1.Locale_LOCALE_UNSPECIFIED {
		apiReq.SetDefaultLocale(v)
	}

	if !data.AuthEmailDomains.IsNull() && !data.AuthEmailDomains.IsUnknown() {
		var domains []string
		resp.Diagnostics.Append(data.AuthEmailDomains.ElementsAs(ctx, &domains, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiReq.SetAuthEmailDomains(domains)
	}

	if !data.Locales.IsNull() && !data.Locales.IsUnknown() {
		var tfLocales []string
		resp.Diagnostics.Append(data.Locales.ElementsAs(ctx, &tfLocales, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiLocales := make([]statuspagev1.Locale, 0, len(tfLocales))
		for _, l := range tfLocales {
			apiLocales = append(apiLocales, localeToProto(l))
		}
		apiReq.SetLocales(apiLocales)
	}

	if !data.AllowIndex.IsNull() && !data.AllowIndex.IsUnknown() {
		apiReq.SetAllowIndex(data.AllowIndex.ValueBool())
	}

	if theme := customThemeToAPI(ctx, data.CustomTheme, &resp.Diagnostics); theme != nil {
		apiReq.SetCustomTheme(theme)
	}
	if resp.Diagnostics.HasError() {
		return
	}

	createResp, err := r.client.StatusPage.CreateStatusPage(ctx, connect.NewRequest(apiReq))
	if err != nil {
		resp.Diagnostics.AddError("Error creating status page", err.Error())
		return
	}

	pageID := createResp.Msg.GetStatusPage().GetId()
	data.ID = types.StringValue(pageID)

	// Read back full state.
	getReq := &statuspagev1.GetStatusPageRequest{}
	getReq.SetId(pageID)
	getResp, err := r.client.StatusPage.GetStatusPage(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Error reading status page after create", err.Error())
		return
	}

	statusPageAPIToModel(ctx, getResp.Msg.GetStatusPage(), &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *statusPageResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data statusPageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &statuspagev1.GetStatusPageRequest{}
	getReq.SetId(data.ID.ValueString())

	apiResp, err := r.client.StatusPage.GetStatusPage(ctx, connect.NewRequest(getReq))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading status page", err.Error())
		return
	}

	statusPageAPIToModel(ctx, apiResp.Msg.GetStatusPage(), &data, &resp.Diagnostics)
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

	updateReq := &statuspagev1.UpdateStatusPageRequest{}
	updateReq.SetId(data.ID.ValueString())
	updateReq.SetTitle(title)
	updateReq.SetSlug(slug)
	updateReq.SetDescription(desc)
	updateReq.SetHomepageUrl(homepageURL)
	updateReq.SetContactUrl(contactURL)
	updateReq.SetIcon(icon)
	updateReq.SetCustomDomain(customDomain)

	// Password has a min_len=1 validation server-side, only send when set.
	if !data.Password.IsNull() && !data.Password.IsUnknown() {
		updateReq.SetPassword(data.Password.ValueString())
	}

	if !data.AccessType.IsNull() && !data.AccessType.IsUnknown() {
		updateReq.SetAccessType(accessTypeToProto(data.AccessType.ValueString()))
	}
	if !data.AuthEmailDomains.IsNull() && !data.AuthEmailDomains.IsUnknown() {
		var domains []string
		resp.Diagnostics.Append(data.AuthEmailDomains.ElementsAs(ctx, &domains, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		updateReq.SetAuthEmailDomains(domains)
	}
	if !data.AllowedIPRanges.IsNull() && !data.AllowedIPRanges.IsUnknown() {
		updateReq.SetAllowedIpRanges(data.AllowedIPRanges.ValueString())
	}
	if !data.Theme.IsNull() && !data.Theme.IsUnknown() {
		updateReq.SetTheme(themeToProto(data.Theme.ValueString()))
	}
	if !data.DefaultLocale.IsNull() && !data.DefaultLocale.IsUnknown() {
		updateReq.SetDefaultLocale(localeToProto(data.DefaultLocale.ValueString()))
	}
	if !data.Locales.IsNull() && !data.Locales.IsUnknown() {
		var tfLocales []string
		resp.Diagnostics.Append(data.Locales.ElementsAs(ctx, &tfLocales, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		apiLocales := make([]statuspagev1.Locale, 0, len(tfLocales))
		for _, l := range tfLocales {
			apiLocales = append(apiLocales, localeToProto(l))
		}
		updateReq.SetLocales(apiLocales)
	}
	if !data.AllowIndex.IsNull() && !data.AllowIndex.IsUnknown() {
		updateReq.SetAllowIndex(data.AllowIndex.ValueBool())
	}

	if !data.CustomTheme.IsNull() && !data.CustomTheme.IsUnknown() {
		if theme := customThemeToAPI(ctx, data.CustomTheme, &resp.Diagnostics); theme != nil {
			updateReq.SetCustomTheme(theme)
		}
		if resp.Diagnostics.HasError() {
			return
		}
	} else if data.CustomTheme.IsNull() {
		// Clear only when state had one: the field's presence triggers the
		// server's plan-feature check, breaking unrelated updates otherwise.
		var state statusPageModel
		resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if !state.CustomTheme.IsNull() {
			updateReq.SetCustomTheme(&statuspagev1.CustomTheme{})
		}
	}

	apiResp, err := r.client.StatusPage.UpdateStatusPage(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Error updating status page", err.Error())
		return
	}

	statusPageAPIToModel(ctx, apiResp.Msg.GetStatusPage(), &data, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *statusPageResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data statusPageModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &statuspagev1.DeleteStatusPageRequest{}
	deleteReq.SetId(data.ID.ValueString())

	_, err := r.client.StatusPage.DeleteStatusPage(ctx, connect.NewRequest(deleteReq))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting status page", err.Error())
	}
}

func (r *statusPageResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), types.StringValue(req.ID))...)
}

func accessTypeToProto(tf string) statuspagev1.PageAccessType {
	switch tf {
	case "public":
		return statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PUBLIC
	case "password":
		return statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PASSWORD_PROTECTED
	case "email-domain":
		return statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_AUTHENTICATED
	case "ip":
		return statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_IP_RESTRICTED
	}
	return statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_UNSPECIFIED
}

func accessTypeFromProto(proto statuspagev1.PageAccessType) (string, bool) {
	switch proto {
	case statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PUBLIC:
		return "public", true
	case statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PASSWORD_PROTECTED:
		return "password", true
	case statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_AUTHENTICATED:
		return "email-domain", true
	case statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_IP_RESTRICTED:
		return "ip", true
	}
	return "", false
}

func themeToProto(tf string) statuspagev1.PageTheme {
	switch tf {
	case "system":
		return statuspagev1.PageTheme_PAGE_THEME_SYSTEM
	case "light":
		return statuspagev1.PageTheme_PAGE_THEME_LIGHT
	case "dark":
		return statuspagev1.PageTheme_PAGE_THEME_DARK
	}
	return statuspagev1.PageTheme_PAGE_THEME_UNSPECIFIED
}

func themeFromProto(proto statuspagev1.PageTheme) (string, bool) {
	switch proto {
	case statuspagev1.PageTheme_PAGE_THEME_SYSTEM:
		return "system", true
	case statuspagev1.PageTheme_PAGE_THEME_LIGHT:
		return "light", true
	case statuspagev1.PageTheme_PAGE_THEME_DARK:
		return "dark", true
	}
	return "", false
}

func localeToProto(tf string) statuspagev1.Locale {
	switch tf {
	case "en":
		return statuspagev1.Locale_LOCALE_EN
	case "fr":
		return statuspagev1.Locale_LOCALE_FR
	case "de":
		return statuspagev1.Locale_LOCALE_DE
	case "hi":
		return statuspagev1.Locale_LOCALE_HI
	case "ko":
		return statuspagev1.Locale_LOCALE_KO
	case "tr":
		return statuspagev1.Locale_LOCALE_TR
	}
	return statuspagev1.Locale_LOCALE_UNSPECIFIED
}

func localeFromProto(proto statuspagev1.Locale) (string, bool) {
	switch proto {
	case statuspagev1.Locale_LOCALE_EN:
		return "en", true
	case statuspagev1.Locale_LOCALE_FR:
		return "fr", true
	case statuspagev1.Locale_LOCALE_DE:
		return "de", true
	case statuspagev1.Locale_LOCALE_HI:
		return "hi", true
	case statuspagev1.Locale_LOCALE_KO:
		return "ko", true
	case statuspagev1.Locale_LOCALE_TR:
		return "tr", true
	}
	return "", false
}

// statusPageAPIToModel maps an API response to the Terraform model.
func statusPageAPIToModel(ctx context.Context, api *statuspagev1.StatusPage, data *statusPageModel, diags *diag.Diagnostics) {
	data.ID = types.StringValue(api.GetId())
	data.Title = types.StringValue(api.GetTitle())
	data.Slug = types.StringValue(api.GetSlug())
	if api.GetDescription() != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(api.GetDescription())
	}
	if api.GetHomepageUrl() != "" || !data.HomepageURL.IsNull() {
		data.HomepageURL = types.StringValue(api.GetHomepageUrl())
	}
	if api.GetContactUrl() != "" || !data.ContactURL.IsNull() {
		data.ContactURL = types.StringValue(api.GetContactUrl())
	}
	if api.GetIcon() != "" || !data.Icon.IsNull() {
		data.Icon = types.StringValue(api.GetIcon())
	}
	if api.GetCustomDomain() != "" || !data.CustomDomain.IsNull() {
		data.CustomDomain = types.StringValue(api.GetCustomDomain())
	}
	data.Published = types.BoolValue(api.GetPublished())
	if v, ok := accessTypeFromProto(api.GetAccessType()); ok {
		data.AccessType = types.StringValue(v)
	} else if api.GetAccessType() != statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_UNSPECIFIED {
		diags.AddWarning(
			"Unknown status page access type",
			fmt.Sprintf("OpenStatus returned access_type %q which this provider version does not recognize.", api.GetAccessType()),
		)
	}
	if api.GetPassword() != "" {
		data.Password = types.StringValue(api.GetPassword())
	}
	if len(api.GetAuthEmailDomains()) > 0 || !data.AuthEmailDomains.IsNull() {
		list, listDiags := types.ListValueFrom(ctx, types.StringType, api.GetAuthEmailDomains())
		diags.Append(listDiags...)
		data.AuthEmailDomains = list
	}
	if v, ok := themeFromProto(api.GetTheme()); ok {
		data.Theme = types.StringValue(v)
	} else if api.GetTheme() != statuspagev1.PageTheme_PAGE_THEME_UNSPECIFIED {
		diags.AddWarning(
			"Unknown status page theme",
			fmt.Sprintf("OpenStatus returned theme %q which this provider version does not recognize.", api.GetTheme()),
		)
	}
	data.CustomTheme = customThemeFromAPI(ctx, api.GetCustomTheme(), diags)
	if v, ok := localeFromProto(api.GetDefaultLocale()); ok {
		data.DefaultLocale = types.StringValue(v)
	} else if api.GetDefaultLocale() != statuspagev1.Locale_LOCALE_UNSPECIFIED {
		diags.AddWarning(
			"Unknown status page default locale",
			fmt.Sprintf("OpenStatus returned default_locale %q which this provider version does not recognize.", api.GetDefaultLocale()),
		)
	}
	if len(api.GetLocales()) > 0 || !data.Locales.IsNull() {
		tfLocales := make([]string, 0, len(api.GetLocales()))
		for _, l := range api.GetLocales() {
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
	data.AllowIndex = types.BoolValue(api.GetAllowIndex())
	if api.GetAllowedIpRanges() != "" || !data.AllowedIPRanges.IsNull() {
		data.AllowedIPRanges = types.StringValue(api.GetAllowedIpRanges())
	}
	data.CreatedAt = types.StringValue(api.GetCreatedAt())
	data.UpdatedAt = types.StringValue(api.GetUpdatedAt())
}
