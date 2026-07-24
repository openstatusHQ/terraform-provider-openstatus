package privatelocation

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	privatelocationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*privateLocationsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*privateLocationsDataSource)(nil)
)

func NewPrivateLocationsDataSource() datasource.DataSource {
	return &privateLocationsDataSource{}
}

type privateLocationsDataSource struct {
	client *client.Client
}

type privateLocationsDataSourceModel struct {
	PrivateLocations types.List  `tfsdk:"private_locations"`
	Limit            types.Int64 `tfsdk:"limit"`
	Offset           types.Int64 `tfsdk:"offset"`
}

var privateLocationSummaryObjTypes = map[string]attr.Type{
	"id":            types.StringType,
	"name":          types.StringType,
	"monitor_count": types.Int64Type,
	"metadata":      types.MapType{ElemType: types.StringType},
	"status":        types.StringType,
	"last_seen_at":  types.StringType,
	"created_at":    types.StringType,
	"updated_at":    types.StringType,
}

func (d *privateLocationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_private_locations"
}

func (d *privateLocationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(providerConfig).Client
}

func (d *privateLocationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists private locations.\n\n" +
			"Agent tokens are not included; use the `openstatus_private_location` data source to fetch one by ID.",
		Attributes: map[string]schema.Attribute{
			"limit": schema.Int64Attribute{
				Optional:            true,
				MarkdownDescription: "Max results (default 50).",
			},
			"offset": schema.Int64Attribute{Optional: true},
			"private_locations": schema.ListNestedAttribute{
				Computed: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
						"monitor_count": schema.Int64Attribute{
							Computed:            true,
							MarkdownDescription: "Number of monitors this private location runs.",
						},
						"metadata": schema.MapAttribute{
							Computed:    true,
							ElementType: types.StringType,
						},
						"status":       schema.StringAttribute{Computed: true},
						"last_seen_at": schema.StringAttribute{Computed: true},
						"created_at":   schema.StringAttribute{Computed: true},
						"updated_at":   schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *privateLocationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data privateLocationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	limit := data.Limit.ValueInt64()
	if limit == 0 {
		limit = 50
	}

	apiReq := &privatelocationv1.ListPrivateLocationsRequest{}
	apiReq.SetLimit(int32(limit))
	apiReq.SetOffset(int32(data.Offset.ValueInt64()))

	apiResp, err := d.client.PrivateLocation.ListPrivateLocations(ctx, connect.NewRequest(apiReq))
	if err != nil {
		resp.Diagnostics.AddError("Error listing private locations", err.Error())
		return
	}

	objs := make([]attr.Value, 0, len(apiResp.Msg.GetPrivateLocations()))
	for _, l := range apiResp.Msg.GetPrivateLocations() {
		metadata, diags := types.MapValueFrom(ctx, types.StringType, l.GetMetadata())
		resp.Diagnostics.Append(diags...)

		obj, diags := types.ObjectValue(privateLocationSummaryObjTypes, map[string]attr.Value{
			"id":            types.StringValue(l.GetId()),
			"name":          types.StringValue(l.GetName()),
			"monitor_count": types.Int64Value(int64(l.GetMonitorCount())),
			"metadata":      metadata,
			"status":        types.StringValue(MapStatusFromAPI(l.GetStatus())),
			"last_seen_at":  optionalTimestamp(l.GetLastSeenAt()),
			"created_at":    types.StringValue(l.GetCreatedAt()),
			"updated_at":    types.StringValue(l.GetUpdatedAt()),
		})
		resp.Diagnostics.Append(diags...)
		objs = append(objs, obj)
	}

	list, diags := types.ListValue(types.ObjectType{AttrTypes: privateLocationSummaryObjTypes}, objs)
	resp.Diagnostics.Append(diags...)
	data.PrivateLocations = list

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
