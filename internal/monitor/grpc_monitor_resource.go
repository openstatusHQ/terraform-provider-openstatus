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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*grpcMonitorResource)(nil)
	_ resource.ResourceWithConfigure   = (*grpcMonitorResource)(nil)
	_ resource.ResourceWithImportState = (*grpcMonitorResource)(nil)
)

func NewGRPCMonitorResource() resource.Resource {
	return &grpcMonitorResource{}
}

type grpcMonitorResource struct {
	client *client.Client
}

type grpcMonitorModel struct {
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
	Service            types.String `tfsdk:"service"`
	TLSMode            types.String `tfsdk:"tls_mode"`
	Metadata           types.List   `tfsdk:"metadata"`
}

func (r *grpcMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_grpc_monitor"
}

func (r *grpcMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(providerConfig).Client
}

func (r *grpcMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a gRPC monitor. The probe calls `grpc.health.v1.Health/Check` on the target.",
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
				MarkdownDescription: "Target in host:port format. IPv6 addresses must be bracketed.",
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
			"service": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Service name passed to `Health/Check`. Leave unset to check overall server health.",
				Validators:          []validator.String{stringvalidator.LengthAtMost(256)},
			},
			"tls_mode": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("tls"),
				MarkdownDescription: "How the connection to the target is secured. One of `tls` (verify the certificate), `plaintext` (h2c) or `tls_insecure` (TLS without certificate verification).",
				Validators:          []validator.String{stringvalidator.OneOf(GRPCTLSModeValues...)},
			},
			"private_location_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "IDs of the private locations that run this monitor. Managed from `openstatus_private_location.monitor_ids`.",
			},
		},
		Blocks: map[string]schema.Block{
			"open_telemetry": openTelemetrySchemaBlock(),
			"metadata":       headersSchemaBlock("Metadata sent with the health check request."),
		},
	}
}

func (r *grpcMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data grpcMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := grpcModelToAPI(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &monitorv1.CreateGRPCMonitorRequest{}
	apiReq.SetMonitor(apiObj)

	apiResp, err := r.client.Monitor.CreateGRPCMonitor(ctx, connect.NewRequest(apiReq))
	if err != nil {
		resp.Diagnostics.AddError("Error creating gRPC monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(grpcAPIToModel(ctx, apiResp.Msg.GetMonitor(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *grpcMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data grpcMonitorModel
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
		resp.Diagnostics.AddError("Error reading gRPC monitor", err.Error())
		return
	}

	monitor := apiResp.Msg.GetMonitor().GetGrpc()
	if monitor == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(grpcAPIToModel(ctx, monitor, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *grpcMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data grpcMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state grpcMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := grpcModelToAPI(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &monitorv1.UpdateGRPCMonitorRequest{}
	updateReq.SetId(state.ID.ValueString())
	updateReq.SetMonitor(apiObj)

	apiResp, err := r.client.Monitor.UpdateGRPCMonitor(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Error updating gRPC monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(grpcAPIToModel(ctx, apiResp.Msg.GetMonitor(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *grpcMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data grpcMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &monitorv1.DeleteMonitorRequest{}
	deleteReq.SetId(data.ID.ValueString())

	_, err := r.client.Monitor.DeleteMonitor(ctx, connect.NewRequest(deleteReq))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting gRPC monitor", err.Error())
	}
}

func (r *grpcMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath, types.StringValue(req.ID))...)
}

func grpcModelToAPI(ctx context.Context, data grpcMonitorModel) (*monitorv1.GRPCMonitor, diag.Diagnostics) {
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

	tlsMode, err := MapGRPCTLSModeToAPI(data.TLSMode.ValueString())
	if err != nil {
		diags.AddError("Invalid tls_mode", err.Error())
		return nil, diags
	}

	metadata, metadataDiags := headersToAPI(ctx, data.Metadata)
	diags.Append(metadataDiags...)
	if diags.HasError() {
		return nil, diags
	}

	otel, otelDiags := openTelemetryToAPI(ctx, data.OpenTelemetry)
	diags.Append(otelDiags...)
	if diags.HasError() {
		return nil, diags
	}

	out := &monitorv1.GRPCMonitor{}
	out.SetName(data.Name.ValueString())
	out.SetUri(data.URI.ValueString())
	out.SetPeriodicity(periodicity)
	out.SetTimeout(data.Timeout.ValueInt64())
	out.SetRetry(data.Retry.ValueInt64())
	out.SetActive(data.Active.ValueBool())
	out.SetPublic(data.Public.ValueBool())
	out.SetDescription(data.Description.ValueString())
	out.SetRegions(regions)
	out.SetService(data.Service.ValueString())
	out.SetTlsMode(tlsMode)
	out.SetMetadata(metadata)
	if otel != nil {
		out.SetOpenTelemetry(otel)
	}
	if v := data.DegradedAt.ValueInt64(); v != 0 {
		out.SetDegradedAt(v)
	}
	return out, diags
}

func grpcAPIToModel(ctx context.Context, api *monitorv1.GRPCMonitor, data *grpcMonitorModel) diag.Diagnostics {
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
	if api.GetService() != "" || !data.Service.IsNull() {
		data.Service = types.StringValue(api.GetService())
	}
	// Unspecified means the API applied its default, which is TLS.
	if api.GetTlsMode() == monitorv1.GRPCTlsMode_GRPC_TLS_MODE_UNSPECIFIED {
		data.TLSMode = types.StringValue("tls")
	} else {
		data.TLSMode = types.StringValue(MapGRPCTLSModeFromAPI(api.GetTlsMode()))
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

	metadataList, d := headersFromAPI(api.GetMetadata())
	diags.Append(d...)
	data.Metadata = metadataList

	otelObj, otelDiags := openTelemetryFromAPI(ctx, api.GetOpenTelemetry(), data.OpenTelemetry)
	diags.Append(otelDiags...)
	data.OpenTelemetry = otelObj

	return diags
}
