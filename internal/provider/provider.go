package provider

import (
	"context"
	"os"

	"terraform-provider-openstatus/internal/client"
	"terraform-provider-openstatus/internal/monitor"
	"terraform-provider-openstatus/internal/notification"
	"terraform-provider-openstatus/internal/privatelocation"
	"terraform-provider-openstatus/internal/statuspage"

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

type openstatusProvider struct{}

type openstatusProviderModel struct {
	APIToken types.String `tfsdk:"api_token"`
	BaseURL  types.String `tfsdk:"base_url"`
}

func (p *openstatusProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "The OpenStatus provider allows you to manage monitors, status pages, notifications, and private locations.",
		Attributes: map[string]schema.Attribute{
			"api_token": schema.StringAttribute{
				MarkdownDescription: "OpenStatus API token. Can also be set via the `OPENSTATUS_API_TOKEN` environment variable.",
				Optional:            true,
				Sensitive:           true,
			},
			"base_url": schema.StringAttribute{
				MarkdownDescription: "Base URL for the OpenStatus API. Defaults to `https://api.openstatus.dev/rpc`.",
				Optional:            true,
			},
		},
	}
}

func (p *openstatusProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data openstatusProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiToken := data.APIToken.ValueString()
	if apiToken == "" {
		apiToken = os.Getenv("OPENSTATUS_API_TOKEN")
	}
	if apiToken == "" {
		resp.Diagnostics.AddError(
			"Missing API Token",
			"Set api_token in the provider configuration or the OPENSTATUS_API_TOKEN environment variable.",
		)
		return
	}

	baseURL := data.BaseURL.ValueString()

	c := client.New(baseURL, apiToken)

	config := client.ProviderConfig{Client: c}
	resp.ResourceData = config
	resp.DataSourceData = config
}

func (p *openstatusProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "openstatus"
}

func (p *openstatusProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		monitor.NewMonitorDataSource,
		monitor.NewMonitorsDataSource,
		statuspage.NewStatusPageDataSource,
		notification.NewNotificationDataSource,
		privatelocation.NewPrivateLocationDataSource,
		privatelocation.NewPrivateLocationsDataSource,
	}
}

func (p *openstatusProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		monitor.NewHTTPMonitorResource,
		monitor.NewTCPMonitorResource,
		monitor.NewDNSMonitorResource,
		monitor.NewICMPMonitorResource,
		notification.NewNotificationResource,
		statuspage.NewStatusPageResource,
		statuspage.NewComponentResource,
		statuspage.NewComponentGroupResource,
		privatelocation.NewPrivateLocationResource,
	}
}
