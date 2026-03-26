package statuspage

import (
	"context"

	"terraform-provider-openstatus/internal/client"

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
	Theme            types.String `tfsdk:"theme"`
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
			"id":                schema.StringAttribute{Required: true},
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
			"theme":              schema.StringAttribute{Computed: true},
			"created_at":         schema.StringAttribute{Computed: true},
			"updated_at":         schema.StringAttribute{Computed: true},
		},
	}
}

func (d *statusPageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data statusPageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResp apiStatusPageResponse
	err := d.client.Do(ctx, "/openstatus.status_page.v1.StatusPageService/GetStatusPage",
		map[string]string{"id": data.ID.ValueString()}, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error reading status page", err.Error())
		return
	}

	api := apiResp.StatusPage
	data.ID = types.StringValue(api.ID)
	data.Title = types.StringValue(api.Title)
	data.Slug = types.StringValue(api.Slug)
	data.Description = types.StringValue(api.Description)
	data.HomepageURL = types.StringValue(api.HomepageURL)
	data.ContactURL = types.StringValue(api.ContactURL)
	data.Icon = types.StringValue(api.Icon)
	data.CustomDomain = types.StringValue(api.CustomDomain)
	data.Published = types.BoolValue(api.Published)
	data.AccessType = types.StringValue(accessTypeFromProto(api.AccessType))
	data.Password = types.StringValue(api.Password)
	list, listDiags := types.ListValueFrom(ctx, types.StringType, api.AuthEmailDomains)
	resp.Diagnostics.Append(listDiags...)
	data.AuthEmailDomains = list
	data.Theme = types.StringValue(themeFromProto(api.Theme))
	data.CreatedAt = types.StringValue(api.CreatedAt)
	data.UpdatedAt = types.StringValue(api.UpdatedAt)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
