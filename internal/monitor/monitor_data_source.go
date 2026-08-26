package monitor

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"connectrpc.com/connect"

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
	ID                 types.String `tfsdk:"id"`
	Type               types.String `tfsdk:"type"`
	Name               types.String `tfsdk:"name"`
	URL                types.String `tfsdk:"url"`
	URI                types.String `tfsdk:"uri"`
	Periodicity        types.String `tfsdk:"periodicity"`
	Method             types.String `tfsdk:"method"`
	Active             types.Bool   `tfsdk:"active"`
	Public             types.Bool   `tfsdk:"public"`
	Description        types.String `tfsdk:"description"`
	Timeout            types.Int64  `tfsdk:"timeout"`
	Status             types.String `tfsdk:"status"`
	PrivateLocationIDs types.Set    `tfsdk:"private_location_ids"`
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
			"private_location_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "IDs of the private locations that run this monitor.",
			},
		},
	}
}

func (d *monitorDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data monitorDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &monitorv1.GetMonitorRequest{}
	getReq.SetId(data.ID.ValueString())

	apiResp, err := d.client.Monitor.GetMonitor(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Error reading monitor", err.Error())
		return
	}

	config := apiResp.Msg.GetMonitor()
	switch {
	case config.GetHttp() != nil:
		m := config.GetHttp()
		data.Type = types.StringValue("http")
		data.Name = types.StringValue(m.GetName())
		data.URL = types.StringValue(m.GetUrl())
		data.Periodicity = types.StringValue(MapPeriodicityFromAPI(m.GetPeriodicity()))
		data.Method = types.StringValue(MapMethodFromAPI(m.GetMethod()))
		data.Active = types.BoolValue(m.GetActive())
		data.Public = types.BoolValue(m.GetPublic())
		data.Description = types.StringValue(m.GetDescription())
		data.Timeout = types.Int64Value(m.GetTimeout())
		if m.GetStatus() != monitorv1.MonitorStatus_MONITOR_STATUS_UNSPECIFIED {
			data.Status = types.StringValue(MapMonitorStatusFromAPI(m.GetStatus()))
		}
	case config.GetTcp() != nil:
		m := config.GetTcp()
		data.Type = types.StringValue("tcp")
		data.Name = types.StringValue(m.GetName())
		data.URI = types.StringValue(m.GetUri())
		data.Periodicity = types.StringValue(MapPeriodicityFromAPI(m.GetPeriodicity()))
		data.Active = types.BoolValue(m.GetActive())
		data.Public = types.BoolValue(m.GetPublic())
		data.Description = types.StringValue(m.GetDescription())
		data.Timeout = types.Int64Value(m.GetTimeout())
		if m.GetStatus() != monitorv1.MonitorStatus_MONITOR_STATUS_UNSPECIFIED {
			data.Status = types.StringValue(MapMonitorStatusFromAPI(m.GetStatus()))
		}
	case config.GetDns() != nil:
		m := config.GetDns()
		data.Type = types.StringValue("dns")
		data.Name = types.StringValue(m.GetName())
		data.URI = types.StringValue(m.GetUri())
		data.Periodicity = types.StringValue(MapPeriodicityFromAPI(m.GetPeriodicity()))
		data.Active = types.BoolValue(m.GetActive())
		data.Public = types.BoolValue(m.GetPublic())
		data.Description = types.StringValue(m.GetDescription())
		data.Timeout = types.Int64Value(m.GetTimeout())
		if m.GetStatus() != monitorv1.MonitorStatus_MONITOR_STATUS_UNSPECIFIED {
			data.Status = types.StringValue(MapMonitorStatusFromAPI(m.GetStatus()))
		}
	case config.GetIcmp() != nil:
		m := config.GetIcmp()
		data.Type = types.StringValue("icmp")
		data.Name = types.StringValue(m.GetName())
		data.URI = types.StringValue(m.GetUri())
		data.Periodicity = types.StringValue(MapPeriodicityFromAPI(m.GetPeriodicity()))
		data.Active = types.BoolValue(m.GetActive())
		data.Public = types.BoolValue(m.GetPublic())
		data.Description = types.StringValue(m.GetDescription())
		data.Timeout = types.Int64Value(m.GetTimeout())
		if m.GetStatus() != monitorv1.MonitorStatus_MONITOR_STATUS_UNSPECIFIED {
			data.Status = types.StringValue(MapMonitorStatusFromAPI(m.GetStatus()))
		}
	default:
		resp.Diagnostics.AddError("Monitor not found", "No monitor returned for ID: "+data.ID.ValueString())
		return
	}

	privateLocationSet, diags := types.SetValueFrom(ctx, types.StringType, privateLocationIDs(config))
	resp.Diagnostics.Append(diags...)
	data.PrivateLocationIDs = privateLocationSet

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func privateLocationIDs(config *monitorv1.MonitorConfig) []string {
	switch {
	case config.GetHttp() != nil:
		return config.GetHttp().GetPrivateLocationIds()
	case config.GetTcp() != nil:
		return config.GetTcp().GetPrivateLocationIds()
	case config.GetDns() != nil:
		return config.GetDns().GetPrivateLocationIds()
	case config.GetIcmp() != nil:
		return config.GetIcmp().GetPrivateLocationIds()
	}
	return nil
}
