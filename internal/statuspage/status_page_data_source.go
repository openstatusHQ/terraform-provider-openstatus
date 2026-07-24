package statuspage

import (
	"context"
	"fmt"

	"terraform-provider-openstatus/internal/client"

	statuspagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*statusPageDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*statusPageDataSource)(nil)
)

func NewStatusPageDataSource() datasource.DataSource {
	return &statusPageDataSource{}
}

type statusPageDataSource struct {
	client *client.Client
}

type statusPageDataSourceModel struct {
	ID               types.String `tfsdk:"id"`
	Title            types.String `tfsdk:"title"`
	Slug             types.String `tfsdk:"slug"`
	Description      types.String `tfsdk:"description"`
	HomepageURL      types.String `tfsdk:"homepage_url"`
	ContactURL       types.String `tfsdk:"contact_url"`
	Icon             types.String `tfsdk:"icon"`
	CustomDomain     types.String `tfsdk:"custom_domain"`
	Published        types.Bool   `tfsdk:"published"`
	AccessType       types.String `tfsdk:"access_type"`
	Password         types.String `tfsdk:"password"`
	AuthEmailDomains types.List   `tfsdk:"auth_email_domains"`
	AllowedIPRanges  types.String `tfsdk:"allowed_ip_ranges"`
	Theme            types.String `tfsdk:"theme"`
	CustomTheme      types.Object `tfsdk:"custom_theme"`
	DefaultLocale    types.String `tfsdk:"default_locale"`
	Locales          types.List   `tfsdk:"locales"`
	AllowIndex       types.Bool   `tfsdk:"allow_index"`
	CreatedAt        types.String `tfsdk:"created_at"`
	UpdatedAt        types.String `tfsdk:"updated_at"`
}

func (d *statusPageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_status_page"
}

func (d *statusPageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(providerConfig).Client
}

func (d *statusPageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an existing status page by ID.",
		Attributes: map[string]schema.Attribute{
			"id":                 schema.StringAttribute{Required: true},
			"title":              schema.StringAttribute{Computed: true},
			"slug":               schema.StringAttribute{Computed: true},
			"description":        schema.StringAttribute{Computed: true},
			"homepage_url":       schema.StringAttribute{Computed: true},
			"contact_url":        schema.StringAttribute{Computed: true},
			"icon":               schema.StringAttribute{Computed: true},
			"custom_domain":      schema.StringAttribute{Computed: true},
			"published":          schema.BoolAttribute{Computed: true},
			"access_type":        schema.StringAttribute{Computed: true},
			"password":           schema.StringAttribute{Computed: true, Sensitive: true},
			"auth_email_domains": schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"allowed_ip_ranges":  schema.StringAttribute{Computed: true},
			"theme":              schema.StringAttribute{Computed: true},
			"custom_theme": schema.SingleNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Per-mode CSS variable overrides merged over `theme`. Null when not configured.",
				Attributes: map[string]schema.Attribute{
					"light": schema.MapAttribute{Computed: true, ElementType: types.StringType},
					"dark":  schema.MapAttribute{Computed: true, ElementType: types.StringType},
				},
			},
			"default_locale": schema.StringAttribute{Computed: true},
			"locales":        schema.ListAttribute{Computed: true, ElementType: types.StringType},
			"allow_index":    schema.BoolAttribute{Computed: true},
			"created_at":     schema.StringAttribute{Computed: true},
			"updated_at":     schema.StringAttribute{Computed: true},
		},
	}
}

func (d *statusPageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data statusPageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &statuspagev1.GetStatusPageRequest{}
	getReq.SetId(data.ID.ValueString())

	apiResp, err := d.client.StatusPage.GetStatusPage(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Error reading status page", err.Error())
		return
	}

	api := apiResp.Msg.GetStatusPage()
	data.ID = types.StringValue(api.GetId())
	data.Title = types.StringValue(api.GetTitle())
	data.Slug = types.StringValue(api.GetSlug())
	data.Description = types.StringValue(api.GetDescription())
	data.HomepageURL = types.StringValue(api.GetHomepageUrl())
	data.ContactURL = types.StringValue(api.GetContactUrl())
	data.Icon = types.StringValue(api.GetIcon())
	data.CustomDomain = types.StringValue(api.GetCustomDomain())
	data.Published = types.BoolValue(api.GetPublished())
	if v, ok := accessTypeFromProto(api.GetAccessType()); ok {
		data.AccessType = types.StringValue(v)
	} else if api.GetAccessType() != statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_UNSPECIFIED {
		resp.Diagnostics.AddWarning(
			"Unknown status page access type",
			fmt.Sprintf("OpenStatus returned access_type %q which this provider version does not recognize.", api.GetAccessType()),
		)
	}
	data.Password = types.StringValue(api.GetPassword())
	list, listDiags := types.ListValueFrom(ctx, types.StringType, api.GetAuthEmailDomains())
	resp.Diagnostics.Append(listDiags...)
	data.AuthEmailDomains = list
	if v, ok := themeFromProto(api.GetTheme()); ok {
		data.Theme = types.StringValue(v)
	} else if api.GetTheme() != statuspagev1.PageTheme_PAGE_THEME_UNSPECIFIED {
		resp.Diagnostics.AddWarning(
			"Unknown status page theme",
			fmt.Sprintf("OpenStatus returned theme %q which this provider version does not recognize.", api.GetTheme()),
		)
	}
	data.CustomTheme = customThemeFromAPI(ctx, api.GetCustomTheme(), &resp.Diagnostics)
	if v, ok := localeFromProto(api.GetDefaultLocale()); ok {
		data.DefaultLocale = types.StringValue(v)
	} else if api.GetDefaultLocale() != statuspagev1.Locale_LOCALE_UNSPECIFIED {
		resp.Diagnostics.AddWarning(
			"Unknown status page default locale",
			fmt.Sprintf("OpenStatus returned default_locale %q which this provider version does not recognize.", api.GetDefaultLocale()),
		)
	}
	tfLocales := make([]string, 0, len(api.GetLocales()))
	for _, l := range api.GetLocales() {
		if v, ok := localeFromProto(l); ok {
			tfLocales = append(tfLocales, v)
		} else {
			resp.Diagnostics.AddWarning(
				"Unknown status page locale",
				fmt.Sprintf("OpenStatus returned locale %q which this provider version does not recognize.", l),
			)
		}
	}
	localesList, localesDiags := types.ListValueFrom(ctx, types.StringType, tfLocales)
	resp.Diagnostics.Append(localesDiags...)
	data.Locales = localesList
	data.AllowIndex = types.BoolValue(api.GetAllowIndex())
	data.AllowedIPRanges = types.StringValue(api.GetAllowedIpRanges())
	data.CreatedAt = types.StringValue(api.GetCreatedAt())
	data.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
