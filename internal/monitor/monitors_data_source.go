package monitor

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*monitorsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*monitorsDataSource)(nil)
)

func NewMonitorsDataSource() datasource.DataSource {
	return &monitorsDataSource{}
}

type monitorsDataSource struct {
	client *client.Client
}

type monitorsDataSourceModel struct {
	Monitors types.List  `tfsdk:"monitors"`
	Limit    types.Int64 `tfsdk:"limit"`
	Offset   types.Int64 `tfsdk:"offset"`
}

var monitorSummaryObjTypes = map[string]attr.Type{
	"id":   types.StringType,
	"name": types.StringType,
	"type": types.StringType,
}

func (d *monitorsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitors"
}

func (d *monitorsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(providerConfig).Client
}

func (d *monitorsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "List monitors.",
		Attributes: map[string]schema.Attribute{
			"limit": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Max results (1–100, default 50).",
			},
			"offset": schema.Int64Attribute{
				Optional: true,
			},
			"monitors": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *monitorsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data monitorsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	limit := data.Limit.ValueInt64()
	if limit == 0 {
		limit = 50
	}

	apiReq := &monitorv1.ListMonitorsRequest{}
	apiReq.SetLimit(int32(limit))
	apiReq.SetOffset(int32(data.Offset.ValueInt64()))

	apiResp, err := d.client.Monitor.ListMonitors(ctx, connect.NewRequest(apiReq))
	if err != nil {
		resp.Diagnostics.AddError("Error listing monitors", err.Error())
		return
	}

	objs := make([]attr.Value, 0)
	for _, m := range apiResp.Msg.GetHttpMonitors() {
		obj, diags := types.ObjectValue(monitorSummaryObjTypes, map[string]attr.Value{
			"id":   types.StringValue(m.GetId()),
			"name": types.StringValue(m.GetName()),
			"type": types.StringValue("http"),
		})
		resp.Diagnostics.Append(diags...)
		objs = append(objs, obj)
	}
	for _, m := range apiResp.Msg.GetTcpMonitors() {
		obj, diags := types.ObjectValue(monitorSummaryObjTypes, map[string]attr.Value{
			"id":   types.StringValue(m.GetId()),
			"name": types.StringValue(m.GetName()),
			"type": types.StringValue("tcp"),
		})
		resp.Diagnostics.Append(diags...)
		objs = append(objs, obj)
	}
	for _, m := range apiResp.Msg.GetDnsMonitors() {
		obj, diags := types.ObjectValue(monitorSummaryObjTypes, map[string]attr.Value{
			"id":   types.StringValue(m.GetId()),
			"name": types.StringValue(m.GetName()),
			"type": types.StringValue("dns"),
		})
		resp.Diagnostics.Append(diags...)
		objs = append(objs, obj)
	}

	monitorList, diags := types.ListValue(types.ObjectType{AttrTypes: monitorSummaryObjTypes}, objs)
	resp.Diagnostics.Append(diags...)
	data.Monitors = monitorList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
