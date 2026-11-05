package monitor

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	heartbeatMinPeriod = 60
	heartbeatMaxPeriod = 2592000 // 30 days
	heartbeatMinGrace  = 60
	heartbeatMaxGrace  = 604800 // 7 days
)

var (
	_ resource.Resource                = (*heartbeatMonitorResource)(nil)
	_ resource.ResourceWithConfigure   = (*heartbeatMonitorResource)(nil)
	_ resource.ResourceWithImportState = (*heartbeatMonitorResource)(nil)
)

func NewHeartbeatMonitorResource() resource.Resource {
	return &heartbeatMonitorResource{}
}

type heartbeatMonitorResource struct {
	client *client.Client
}

// heartbeatMonitorModel deliberately leaves out status and the last ping: they
// change with every ping, which would report drift on every plan and can fail
// an apply with an inconsistent result. The openstatus_monitor data source
// exposes them.
type heartbeatMonitorModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Period      types.Int64  `tfsdk:"period"`
	Grace       types.Int64  `tfsdk:"grace"`
	Active      types.Bool   `tfsdk:"active"`
	Public      types.Bool   `tfsdk:"public"`
	Description types.String `tfsdk:"description"`
	PingURL     types.String `tfsdk:"ping_url"`
}

func (r *heartbeatMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_heartbeat_monitor"
}

func (r *heartbeatMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(providerConfig).Client
}

func (r *heartbeatMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a heartbeat monitor for jobs that run on their own schedule, such as cron jobs, queue workers or backups. OpenStatus never calls the job: the job requests `ping_url` each time it runs, and the monitor goes down when a ping is late or the job requests `ping_url` followed by `/fail`.\n\n" +
			"Heartbeat monitors are being rolled out gradually. If creating one fails with \"Heartbeat monitors are not available yet\", contact ping@openstatus.dev to have them enabled for your workspace.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthBetween(1, 256)},
			},
			"period": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Expected time between pings, in seconds. Between 60 (1 minute) and 2592000 (30 days).",
				Validators:          []validator.Int64{int64validator.Between(heartbeatMinPeriod, heartbeatMaxPeriod)},
			},
			"grace": schema.Int64Attribute{
				Required:            true,
				MarkdownDescription: "Extra time after `period` before a late ping counts as missed, in seconds. Between 60 (1 minute) and 604800 (7 days). Set it to cover how long the job runs plus any start-up delay.",
				Validators:          []validator.Int64{int64validator.Between(heartbeatMinGrace, heartbeatMaxGrace)},
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the monitor is active. Pings to an inactive monitor are ignored and it sends no alerts. Defaults to `false`.",
			},
			"public": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
			},
			"description": schema.StringAttribute{
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthAtMost(1024)},
			},
			"ping_url": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "URL the job requests each time it runs. It needs no authentication, so treat it as a secret.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *heartbeatMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data heartbeatMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &monitorv1.CreateHeartbeatMonitorRequest{}
	apiReq.SetMonitor(heartbeatModelToAPI(data))

	apiResp, err := r.client.Monitor.CreateHeartbeatMonitor(ctx, connect.NewRequest(apiReq))
	if err != nil {
		resp.Diagnostics.AddError("Error creating heartbeat monitor", err.Error())
		return
	}

	heartbeatAPIToModel(apiResp.Msg.GetMonitor(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *heartbeatMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data heartbeatMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	getReq := &monitorv1.GetMonitorRequest{}
	getReq.SetId(data.ID.ValueString())

	apiResp, err := r.client.Monitor.GetMonitor(ctx, connect.NewRequest(getReq))
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading heartbeat monitor", err.Error())
		return
	}

	monitor := apiResp.Msg.GetMonitor().GetHeartbeat()
	if monitor == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	heartbeatAPIToModel(monitor, &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *heartbeatMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data heartbeatMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state heartbeatMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &monitorv1.UpdateHeartbeatMonitorRequest{}
	updateReq.SetId(state.ID.ValueString())
	updateReq.SetMonitor(heartbeatModelToAPI(data))

	apiResp, err := r.client.Monitor.UpdateHeartbeatMonitor(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Error updating heartbeat monitor", err.Error())
		return
	}

	heartbeatAPIToModel(apiResp.Msg.GetMonitor(), &data)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *heartbeatMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data heartbeatMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &monitorv1.DeleteMonitorRequest{}
	deleteReq.SetId(data.ID.ValueString())

	_, err := r.client.Monitor.DeleteMonitor(ctx, connect.NewRequest(deleteReq))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting heartbeat monitor", err.Error())
	}
}

func (r *heartbeatMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath, types.StringValue(req.ID))...)
}

// heartbeatModelToAPI always sets description, active and public: the API only
// applies a field that is present on update, so leaving one out when it is
// removed from the configuration would keep the old value.
func heartbeatModelToAPI(data heartbeatMonitorModel) *monitorv1.HeartbeatMonitor {
	out := &monitorv1.HeartbeatMonitor{}
	out.SetName(data.Name.ValueString())
	out.SetPeriod(data.Period.ValueInt64())
	out.SetGrace(data.Grace.ValueInt64())
	out.SetActive(data.Active.ValueBool())
	out.SetPublic(data.Public.ValueBool())
	out.SetDescription(data.Description.ValueString())
	return out
}

func heartbeatAPIToModel(api *monitorv1.HeartbeatMonitor, data *heartbeatMonitorModel) {
	data.ID = types.StringValue(api.GetId())
	data.Name = types.StringValue(api.GetName())
	data.Period = types.Int64Value(api.GetPeriod())
	data.Grace = types.Int64Value(api.GetGrace())
	data.Active = types.BoolValue(api.GetActive())
	data.Public = types.BoolValue(api.GetPublic())
	if api.GetDescription() != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(api.GetDescription())
	}
	data.PingURL = types.StringValue(api.GetPingUrl())
}
