package monitor

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*tcpMonitorResource)(nil)
	_ resource.ResourceWithConfigure   = (*tcpMonitorResource)(nil)
	_ resource.ResourceWithImportState = (*tcpMonitorResource)(nil)
)

func NewTCPMonitorResource() resource.Resource {
	return &tcpMonitorResource{}
}

type tcpMonitorResource struct {
	client *client.Client
}

type tcpMonitorModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	URI                types.String `tfsdk:"uri"`
	Periodicity        types.String `tfsdk:"periodicity"`
	Timeout            types.Int64  `tfsdk:"timeout"`
	DegradedAt         types.Int64  `tfsdk:"degraded_at"`
	Retry              types.Int64  `tfsdk:"retry"`
	Active             types.Bool   `tfsdk:"active"`
	Public             types.Bool   `tfsdk:"public"`
	Description        types.String `tfsdk:"description"`
	Regions            types.Set    `tfsdk:"regions"`
	OpenTelemetry      types.Object `tfsdk:"open_telemetry"`
	Status             types.String `tfsdk:"status"`
	PrivateLocationIDs types.Set    `tfsdk:"private_location_ids"`
}

func (r *tcpMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tcp_monitor"
}

func (r *tcpMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(providerConfig).Client
}

func (r *tcpMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a TCP monitor.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.LengthAtMost(256)},
			},
			"uri": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Target URI in host:port format.",
				Validators:          []validator.String{stringvalidator.LengthAtMost(2048)},
			},
			"periodicity": schema.StringAttribute{
				Required:   true,
				Validators: []validator.String{stringvalidator.OneOf(PeriodicityValues...)},
			},
			"timeout": schema.Int64Attribute{
				Optional:   true,
				Computed:   true,
				Default:    int64default.StaticInt64(45000),
				Validators: []validator.Int64{int64validator.Between(0, 120000)},
			},
			"degraded_at": schema.Int64Attribute{
				Optional:   true,
				Computed:   true,
				Validators: []validator.Int64{int64validator.Between(0, 120000)},
			},
			"retry": schema.Int64Attribute{
				Optional:   true,
				Computed:   true,
				Default:    int64default.StaticInt64(3),
				Validators: []validator.Int64{int64validator.Between(0, 10)},
			},
			"active": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Default:  booldefault.StaticBool(false),
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
			"regions": schema.SetAttribute{
				Optional:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "Regions to monitor from.",
				Validators: []validator.Set{
					setvalidator.ValueStringsAre(stringvalidator.OneOf(RegionValues...)),
				},
			},
			"status": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"private_location_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "IDs of the private locations that run this monitor. Managed from `openstatus_private_location.monitor_ids`.",
			},
		},
		Blocks: map[string]schema.Block{
			"open_telemetry": openTelemetrySchemaBlock(),
		},
	}
}

func (r *tcpMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data tcpMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := tcpModelToAPI(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &monitorv1.CreateTCPMonitorRequest{}
	apiReq.SetMonitor(apiObj)

	apiResp, err := r.client.Monitor.CreateTCPMonitor(ctx, connect.NewRequest(apiReq))
	if err != nil {
		resp.Diagnostics.AddError("Error creating TCP monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(tcpAPIToModel(ctx, apiResp.Msg.GetMonitor(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tcpMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tcpMonitorModel
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
		resp.Diagnostics.AddError("Error reading TCP monitor", err.Error())
		return
	}

	monitor := apiResp.Msg.GetMonitor().GetTcp()
	if monitor == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(tcpAPIToModel(ctx, monitor, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tcpMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data tcpMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state tcpMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := tcpModelToAPI(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &monitorv1.UpdateTCPMonitorRequest{}
	updateReq.SetId(state.ID.ValueString())
	updateReq.SetMonitor(apiObj)

	apiResp, err := r.client.Monitor.UpdateTCPMonitor(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Error updating TCP monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(tcpAPIToModel(ctx, apiResp.Msg.GetMonitor(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tcpMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tcpMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &monitorv1.DeleteMonitorRequest{}
	deleteReq.SetId(data.ID.ValueString())

	_, err := r.client.Monitor.DeleteMonitor(ctx, connect.NewRequest(deleteReq))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting TCP monitor", err.Error())
	}
}

func (r *tcpMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath, types.StringValue(req.ID))...)
}

func tcpModelToAPI(ctx context.Context, data tcpMonitorModel) (*monitorv1.TCPMonitor, diag.Diagnostics) {
	var diags diag.Diagnostics

	periodicity, err := MapPeriodicityToAPI(data.Periodicity.ValueString())
	if err != nil {
		diags.AddError("Invalid periodicity", err.Error())
		return nil, diags
	}

	var regions []monitorv1.Region
	if !data.Regions.IsNull() && !data.Regions.IsUnknown() {
		var tfRegions []string
		diags.Append(data.Regions.ElementsAs(ctx, &tfRegions, false)...)
		if diags.HasError() {
			return nil, diags
		}
		regions, err = MapRegionsToAPI(tfRegions)
		if err != nil {
			diags.AddError("Invalid region", err.Error())
			return nil, diags
		}
	}

	otel, otelDiags := openTelemetryToAPI(ctx, data.OpenTelemetry)
	diags.Append(otelDiags...)
	if diags.HasError() {
		return nil, diags
	}

	out := &monitorv1.TCPMonitor{}
	out.SetName(data.Name.ValueString())
	out.SetUri(data.URI.ValueString())
	out.SetPeriodicity(periodicity)
	out.SetTimeout(data.Timeout.ValueInt64())
	out.SetRetry(data.Retry.ValueInt64())
	out.SetActive(data.Active.ValueBool())
	out.SetPublic(data.Public.ValueBool())
	out.SetDescription(data.Description.ValueString())
	out.SetRegions(regions)
	if otel != nil {
		out.SetOpenTelemetry(otel)
	}
	if v := data.DegradedAt.ValueInt64(); v != 0 {
		out.SetDegradedAt(v)
	}
	return out, diags
}

func tcpAPIToModel(ctx context.Context, api *monitorv1.TCPMonitor, data *tcpMonitorModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(api.GetId())
	data.Name = types.StringValue(api.GetName())
	data.URI = types.StringValue(api.GetUri())
	data.Periodicity = types.StringValue(MapPeriodicityFromAPI(api.GetPeriodicity()))
	data.Timeout = types.Int64Value(api.GetTimeout())
	data.DegradedAt = types.Int64Value(api.GetDegradedAt())
	data.Retry = types.Int64Value(api.GetRetry())
	data.Active = types.BoolValue(api.GetActive())
	data.Public = types.BoolValue(api.GetPublic())
	if api.GetDescription() != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(api.GetDescription())
	}
	data.Status = types.StringValue(MapMonitorStatusFromAPI(api.GetStatus()))

	privateLocationSet, d := types.SetValueFrom(ctx, types.StringType, api.GetPrivateLocationIds())
	diags.Append(d...)
	data.PrivateLocationIDs = privateLocationSet

	if len(api.GetRegions()) > 0 {
		regionVals := MapRegionsFromAPI(api.GetRegions())
		regionSet, d := types.SetValueFrom(ctx, types.StringType, regionVals)
		diags.Append(d...)
		data.Regions = regionSet
	} else if !data.Regions.IsNull() {
		data.Regions = types.SetNull(types.StringType)
	}

	otelObj, otelDiags := openTelemetryFromAPI(ctx, api.GetOpenTelemetry())
	diags.Append(otelDiags...)
	data.OpenTelemetry = otelObj

	return diags
}
