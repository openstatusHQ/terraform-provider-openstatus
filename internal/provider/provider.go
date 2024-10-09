package provider

import (
	"context"

	hreq "github.com/imroc/req/v3"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = (*openstatusProvider)(nil)

func New() func() provider.Provider {
	return func() provider.Provider {
		return &openstatusProvider{}
	}
}

type ProviderConfig struct {
	client *hreq.Client
}

type openstatusProvider struct {
	client *hreq.Client
	token  string
}

type openStatusProviderData struct {
	OpenStatusToken types.String `tfsdk:"openstatus_api_token"`
}

func (p *openstatusProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"openstatus_api_token": schema.StringAttribute{
				MarkdownDescription: "openstatus.dev api token.",
				Required:            true,
			},
		},
	}
}

func (p *openstatusProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data openStatusProviderData
	diags := req.Config.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)

	if resp.Diagnostics.HasError() {
		return
	}
	if data.OpenStatusToken.IsUnknown() || data.OpenStatusToken.IsNull() {
		resp.Diagnostics.AddError("openstatus_api_token is required",
			"openstatus_api_token is required")
		return
	}
	token := data.OpenStatusToken.ValueString()
	p.token = token
	p.client = hreq.C()
	p.client.SetBaseURL("https://api.openstatus.dev/v1/")
	p.client.SetCommonHeader("x-openstatus-key", token)

	resp.ResourceData = ProviderConfig{
		client: p.client,
	}
}

func (p *openstatusProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "openstatus"
}

func (p *openstatusProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}

func (p *openstatusProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewMonitorResource,
	}
}
