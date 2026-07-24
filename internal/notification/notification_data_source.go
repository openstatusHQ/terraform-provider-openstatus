package notification

import (
	"context"
	"fmt"

	"terraform-provider-openstatus/internal/client"

	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*notificationDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*notificationDataSource)(nil)
)

func NewNotificationDataSource() datasource.DataSource {
	return &notificationDataSource{}
}

type notificationDataSource struct {
	client *client.Client
}

type notificationDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	ProviderType types.String `tfsdk:"provider_type"`
	MonitorIDs   types.Set    `tfsdk:"monitor_ids"`
	CreatedAt    types.String `tfsdk:"created_at"`
	UpdatedAt    types.String `tfsdk:"updated_at"`
}

func (d *notificationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification"
}

func (d *notificationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, _ *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	d.client = req.ProviderData.(providerConfig).Client
}

func (d *notificationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Look up an existing notification by ID.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Required: true},
			"name":          schema.StringAttribute{Computed: true},
			"provider_type": schema.StringAttribute{Computed: true},
			"monitor_ids": schema.SetAttribute{
				Computed:    true,
				ElementType: types.StringType,
			},
			"created_at": schema.StringAttribute{Computed: true},
			"updated_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (d *notificationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data notificationDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &notificationv1.GetNotificationRequest{}
	getReq.SetId(data.ID.ValueString())

	apiResp, err := d.client.Notification.GetNotification(ctx, connect.NewRequest(getReq))
	if err != nil {
		resp.Diagnostics.AddError("Error reading notification", err.Error())
		return
	}

	api := apiResp.Msg.GetNotification()
	data.ID = types.StringValue(api.GetId())
	data.Name = types.StringValue(api.GetName())
	data.CreatedAt = types.StringValue(api.GetCreatedAt())
	data.UpdatedAt = types.StringValue(api.GetUpdatedAt())

	if pt, ok := providerTypeFromAPI[api.GetProvider()]; ok {
		data.ProviderType = types.StringValue(pt)
	} else {
		resp.Diagnostics.AddWarning(
			"Unknown notification provider",
			fmt.Sprintf(
				"OpenStatus returned provider %q which this provider version does not recognize. "+
					"State for notification %q may be incomplete; upgrade the openstatus provider to manage this resource.",
				api.GetProvider().String(), api.GetId(),
			),
		)
	}

	if len(api.GetMonitorIds()) > 0 {
		monitorSet, diags := types.SetValueFrom(ctx, types.StringType, api.GetMonitorIds())
		resp.Diagnostics.Append(diags...)
		data.MonitorIDs = monitorSet
	} else {
		data.MonitorIDs = types.SetNull(types.StringType)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
