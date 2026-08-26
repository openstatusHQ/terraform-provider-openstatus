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
	_ resource.Resource                = (*icmpMonitorResource)(nil)
	_ resource.ResourceWithConfigure   = (*icmpMonitorResource)(nil)
	_ resource.ResourceWithImportState = (*icmpMonitorResource)(nil)
)

func NewICMPMonitorResource() resource.Resource {
	return &icmpMonitorResource{}
}

type icmpMonitorResource struct {
	client *client.Client
}

type icmpMonitorModel struct {
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

func (r *icmpMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_icmp_monitor"
}

func (r *icmpMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(providerConfig).Client
}

func (r *icmpMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an ICMP monitor.",
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
				MarkdownDescription: "Target host or IP address to ping.",
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

func (r *icmpMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data icmpMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := icmpModelToAPI(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &monitorv1.CreateICMPMonitorRequest{}
	apiReq.SetMonitor(apiObj)

	apiResp, err := r.client.Monitor.CreateICMPMonitor(ctx, connect.NewRequest(apiReq))
	if err != nil {
		resp.Diagnostics.AddError("Error creating ICMP monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(icmpAPIToModel(ctx, apiResp.Msg.GetMonitor(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *icmpMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data icmpMonitorModel
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
		resp.Diagnostics.AddError("Error reading ICMP monitor", err.Error())
		return
	}

	monitor := apiResp.Msg.GetMonitor().GetIcmp()
	if monitor == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(icmpAPIToModel(ctx, monitor, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *icmpMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data icmpMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state icmpMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := icmpModelToAPI(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &monitorv1.UpdateICMPMonitorRequest{}
	updateReq.SetId(state.ID.ValueString())
	updateReq.SetMonitor(apiObj)

	apiResp, err := r.client.Monitor.UpdateICMPMonitor(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Error updating ICMP monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(icmpAPIToModel(ctx, apiResp.Msg.GetMonitor(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *icmpMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data icmpMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &monitorv1.DeleteMonitorRequest{}
	deleteReq.SetId(data.ID.ValueString())

	_, err := r.client.Monitor.DeleteMonitor(ctx, connect.NewRequest(deleteReq))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting ICMP monitor", err.Error())
	}
}

func (r *icmpMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath, types.StringValue(req.ID))...)
}

func icmpModelToAPI(ctx context.Context, data icmpMonitorModel) (*monitorv1.ICMPMonitor, diag.Diagnostics) {
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

	out := &monitorv1.ICMPMonitor{}
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

func icmpAPIToModel(ctx context.Context, api *monitorv1.ICMPMonitor, data *icmpMonitorModel) diag.Diagnostics {
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

	otelObj, otelDiags := openTelemetryFromAPI(ctx, api.GetOpenTelemetry(), data.OpenTelemetry)
	diags.Append(otelDiags...)
	data.OpenTelemetry = otelObj

	return diags
}
