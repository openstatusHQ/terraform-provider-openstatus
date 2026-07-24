package privatelocation

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	privatelocationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*privateLocationDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*privateLocationDataSource)(nil)
)

func NewPrivateLocationDataSource() datasource.DataSource {
	return &privateLocationDataSource{}
}

type privateLocationDataSource struct {
	client *client.Client
}

type privateLocationDataSourceModel struct {
	ID         types.String `tfsdk:"id"`
	Name       types.String `tfsdk:"name"`
	Token      types.String `tfsdk:"token"`
	MonitorIDs types.Set    `tfsdk:"monitor_ids"`
	Metadata   types.Map    `tfsdk:"metadata"`
	Status     types.String `tfsdk:"status"`
	LastSeenAt types.String `tfsdk:"last_seen_at"`
	CreatedAt  types.String `tfsdk:"created_at"`
	UpdatedAt  types.String `tfsdk:"updated_at"`
}

func (d *privateLocationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_location"
}

func (d *privateLocationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(providerConfig).Client
}

func (d *privateLocationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Fetches a single private location by ID, including its agent token.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "ID of the private location.",
			},
			"name": schema.StringAttribute{Computed: true},
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Agent credential. Treat it as a secret.",
			},
			"monitor_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"metadata": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Computed health of the agent: `active`, `error`, or `unknown`.",
			},
			"last_seen_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "When the agent last reported a result (RFC 3339). Null if it never has.",
			},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *privateLocationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data privateLocationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &privatelocationv1.GetPrivateLocationRequest{}
	getReq.SetId(data.ID.ValueString())

	apiResp, err := d.client.PrivateLocation.GetPrivateLocation(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Error reading private location", err.Error())
		return
	}

	api := apiResp.Msg.GetPrivateLocation()
	data.ID = types.StringValue(api.GetId())
	data.Name = types.StringValue(api.GetName())
	data.Token = types.StringValue(api.GetToken())
	data.Status = types.StringValue(MapStatusFromAPI(api.GetStatus()))
	data.LastSeenAt = optionalTimestamp(api.GetLastSeenAt())
	data.CreatedAt = types.StringValue(api.GetCreatedAt())
	data.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	monitorSet, diags := types.SetValueFrom(ctx, types.StringType, api.GetMonitorIds())
	resp.Diagnostics.Append(diags...)
	data.MonitorIDs = monitorSet

	metadata, diags := types.MapValueFrom(ctx, types.StringType, api.GetMetadata())
	resp.Diagnostics.Append(diags...)
	data.Metadata = metadata

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// optionalTimestamp maps the API's empty-string sentinel to null so a location
// whose agent has never reported does not read as an empty timestamp.
func optionalTimestamp(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

var statusFromAPI = map[privatelocationv1.PrivateLocationStatus]string{
	privatelocationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_ACTIVE: "active",
	privatelocationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_ERROR:  "error",
}

func MapStatusFromAPI(v privatelocationv1.PrivateLocationStatus) string {
	if s, ok := statusFromAPI[v]; ok {
		return s
	}
	return "unknown"
}
