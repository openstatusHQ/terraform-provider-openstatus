package monitor

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*monitorDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*monitorDataSource)(nil)
)

func NewMonitorDataSource() datasource.DataSource {
	return &monitorDataSource{}
}

type monitorDataSource struct {
	client *client.Client
}

type monitorDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Type        types.String `tfsdk:"type"`
	Name        types.String `tfsdk:"name"`
	URL         types.String `tfsdk:"url"`
	URI         types.String `tfsdk:"uri"`
	Periodicity types.String `tfsdk:"periodicity"`
	Method      types.String `tfsdk:"method"`
	Active      types.Bool   `tfsdk:"active"`
	Public      types.Bool   `tfsdk:"public"`
	Description types.String `tfsdk:"description"`
	Timeout     types.Int64  `tfsdk:"timeout"`
	Status      types.String `tfsdk:"status"`
}

func (d *monitorDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (d *monitorDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(providerConfig).Client
}

func (d *monitorDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an existing monitor by ID.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Required: true},
			"type":        schema.StringAttribute{Computed: true},
			"name":        schema.StringAttribute{Computed: true},
			"url":         schema.StringAttribute{Computed: true},
			"uri":         schema.StringAttribute{Computed: true},
			"periodicity": schema.StringAttribute{Computed: true},
			"method":      schema.StringAttribute{Computed: true},
			"active":      schema.BoolAttribute{Computed: true},
			"public":      schema.BoolAttribute{Computed: true},
			"description": schema.StringAttribute{Computed: true},
			"timeout":     schema.Int64Attribute{Computed: true},
			"status":      schema.StringAttribute{Computed: true},
		},
	}
}

func (d *monitorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data monitorDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResp getMonitorResponse
	err := d.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/GetMonitor",
		map[string]string{"id": data.ID.ValueString()}, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error reading monitor", err.Error())
		return
	}

	switch {
	case apiResp.HTTP != nil:
		data.Type = types.StringValue("http")
		data.Name = types.StringValue(apiResp.HTTP.Name)
		data.URL = types.StringValue(apiResp.HTTP.URL)
		data.Periodicity = types.StringValue(MapPeriodicityFromAPI(apiResp.HTTP.Periodicity))
		data.Method = types.StringValue(MapMethodFromAPI(apiResp.HTTP.Method))
		data.Active = types.BoolValue(apiResp.HTTP.Active)
		data.Public = types.BoolValue(apiResp.HTTP.Public)
		data.Description = types.StringValue(apiResp.HTTP.Description)
		data.Timeout = types.Int64Value(apiResp.HTTP.Timeout.Int64())
		if apiResp.HTTP.Status != "" {
			data.Status = types.StringValue(MapMonitorStatusFromAPI(apiResp.HTTP.Status))
		}
	case apiResp.TCP != nil:
		data.Type = types.StringValue("tcp")
		data.Name = types.StringValue(apiResp.TCP.Name)
		data.URI = types.StringValue(apiResp.TCP.URI)
		data.Periodicity = types.StringValue(MapPeriodicityFromAPI(apiResp.TCP.Periodicity))
		data.Active = types.BoolValue(apiResp.TCP.Active)
		data.Public = types.BoolValue(apiResp.TCP.Public)
		data.Description = types.StringValue(apiResp.TCP.Description)
		data.Timeout = types.Int64Value(apiResp.TCP.Timeout.Int64())
		if apiResp.TCP.Status != "" {
			data.Status = types.StringValue(MapMonitorStatusFromAPI(apiResp.TCP.Status))
		}
	case apiResp.DNS != nil:
		data.Type = types.StringValue("dns")
		data.Name = types.StringValue(apiResp.DNS.Name)
		data.URI = types.StringValue(apiResp.DNS.URI)
		data.Periodicity = types.StringValue(MapPeriodicityFromAPI(apiResp.DNS.Periodicity))
		data.Active = types.BoolValue(apiResp.DNS.Active)
		data.Public = types.BoolValue(apiResp.DNS.Public)
		data.Description = types.StringValue(apiResp.DNS.Description)
		data.Timeout = types.Int64Value(apiResp.DNS.Timeout.Int64())
		if apiResp.DNS.Status != "" {
			data.Status = types.StringValue(MapMonitorStatusFromAPI(apiResp.DNS.Status))
		}
	default:
		resp.Diagnostics.AddError("Monitor not found", "No monitor returned for ID: "+data.ID.ValueString())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
