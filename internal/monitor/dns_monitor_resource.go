package monitor

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"connectrpc.com/connect"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
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
	_ resource.Resource                = (*dnsMonitorResource)(nil)
	_ resource.ResourceWithConfigure   = (*dnsMonitorResource)(nil)
	_ resource.ResourceWithImportState = (*dnsMonitorResource)(nil)
)

func NewDNSMonitorResource() resource.Resource {
	return &dnsMonitorResource{}
}

type dnsMonitorResource struct {
	client *client.Client
}

type dnsMonitorModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	URI              types.String `tfsdk:"uri"`
	Periodicity      types.String `tfsdk:"periodicity"`
	Timeout          types.Int64  `tfsdk:"timeout"`
	DegradedAt       types.Int64  `tfsdk:"degraded_at"`
	Retry            types.Int64  `tfsdk:"retry"`
	Active           types.Bool   `tfsdk:"active"`
	Public           types.Bool   `tfsdk:"public"`
	Description      types.String `tfsdk:"description"`
	Regions          types.Set    `tfsdk:"regions"`
	RecordAssertions types.List   `tfsdk:"record_assertions"`
	OpenTelemetry    types.Object `tfsdk:"open_telemetry"`
	Status           types.String `tfsdk:"status"`
}

var recordAssertionObjTypes = map[string]attr.Type{
	"record":     types.StringType,
	"comparator": types.StringType,
	"target":     types.StringType,
}

func (r *dnsMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_dns_monitor"
}

func (r *dnsMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(providerConfig).Client
}

func (r *dnsMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a DNS monitor.",
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
				MarkdownDescription: "Domain name to monitor.",
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
		},
		Blocks: map[string]schema.Block{
			"record_assertions": schema.ListNestedBlock{
				Validators: []validator.List{listvalidator.SizeAtMost(10)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"record": schema.StringAttribute{
							Required:   true,
							Validators: []validator.String{stringvalidator.OneOf(DNSRecordTypes...)},
						},
						"comparator": schema.StringAttribute{
							Required:   true,
							Validators: []validator.String{stringvalidator.OneOf(RecordComparatorValues...)},
						},
						"target": schema.StringAttribute{
							Required: true,
						},
					},
				},
			},
			"open_telemetry": openTelemetrySchemaBlock(),
		},
	}
}

func (r *dnsMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data dnsMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := dnsModelToAPI(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &monitorv1.CreateDNSMonitorRequest{}
	apiReq.SetMonitor(apiObj)

	apiResp, err := r.client.Monitor.CreateDNSMonitor(ctx, connect.NewRequest(apiReq))
	if err != nil {
		resp.Diagnostics.AddError("Error creating DNS monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(dnsAPIToModel(ctx, apiResp.Msg.GetMonitor(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dnsMonitorModel
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
		resp.Diagnostics.AddError("Error reading DNS monitor", err.Error())
		return
	}

	monitor := apiResp.Msg.GetMonitor().GetDns()
	if monitor == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(dnsAPIToModel(ctx, monitor, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data dnsMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state dnsMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := dnsModelToAPI(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &monitorv1.UpdateDNSMonitorRequest{}
	updateReq.SetId(state.ID.ValueString())
	updateReq.SetMonitor(apiObj)

	apiResp, err := r.client.Monitor.UpdateDNSMonitor(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Error updating DNS monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(dnsAPIToModel(ctx, apiResp.Msg.GetMonitor(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data dnsMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &monitorv1.DeleteMonitorRequest{}
	deleteReq.SetId(data.ID.ValueString())

	_, err := r.client.Monitor.DeleteMonitor(ctx, connect.NewRequest(deleteReq))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting DNS monitor", err.Error())
	}
}

func (r *dnsMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath, types.StringValue(req.ID))...)
}

func dnsModelToAPI(ctx context.Context, data dnsMonitorModel) (*monitorv1.DNSMonitor, diag.Diagnostics) {
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

	var recordAssertions []*monitorv1.RecordAssertion
	if !data.RecordAssertions.IsNull() && !data.RecordAssertions.IsUnknown() {
		var tfAssertions []struct {
			Record     string `tfsdk:"record"`
			Comparator string `tfsdk:"comparator"`
			Target     string `tfsdk:"target"`
		}
		diags.Append(data.RecordAssertions.ElementsAs(ctx, &tfAssertions, false)...)
		for _, a := range tfAssertions {
			comp, mapErr := MapRecordComparatorToAPI(a.Comparator)
			if mapErr != nil {
				diags.AddError("Invalid comparator", mapErr.Error())
				return nil, diags
			}
			assertion := &monitorv1.RecordAssertion{}
			assertion.SetRecord(a.Record)
			assertion.SetComparator(comp)
			assertion.SetTarget(a.Target)
			recordAssertions = append(recordAssertions, assertion)
		}
	}

	otel, otelDiags := openTelemetryToAPI(ctx, data.OpenTelemetry)
	diags.Append(otelDiags...)
	if diags.HasError() {
		return nil, diags
	}

	out := &monitorv1.DNSMonitor{}
	out.SetName(data.Name.ValueString())
	out.SetUri(data.URI.ValueString())
	out.SetPeriodicity(periodicity)
	out.SetTimeout(data.Timeout.ValueInt64())
	out.SetRetry(data.Retry.ValueInt64())
	out.SetActive(data.Active.ValueBool())
	out.SetPublic(data.Public.ValueBool())
	out.SetDescription(data.Description.ValueString())
	out.SetRegions(regions)
	out.SetRecordAssertions(recordAssertions)
	if otel != nil {
		out.SetOpenTelemetry(otel)
	}
	if v := data.DegradedAt.ValueInt64(); v != 0 {
		out.SetDegradedAt(v)
	}
	return out, diags
}

func dnsAPIToModel(ctx context.Context, api *monitorv1.DNSMonitor, data *dnsMonitorModel) diag.Diagnostics {
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

	if len(api.GetRegions()) > 0 {
		regionVals := MapRegionsFromAPI(api.GetRegions())
		regionSet, d := types.SetValueFrom(ctx, types.StringType, regionVals)
		diags.Append(d...)
		data.Regions = regionSet
	} else if !data.Regions.IsNull() {
		data.Regions = types.SetNull(types.StringType)
	}

	if len(api.GetRecordAssertions()) > 0 {
		objs := make([]attr.Value, 0, len(api.GetRecordAssertions()))
		for _, a := range api.GetRecordAssertions() {
			obj, d := types.ObjectValue(recordAssertionObjTypes, map[string]attr.Value{
				"record":     types.StringValue(a.GetRecord()),
				"comparator": types.StringValue(MapRecordComparatorFromAPI(a.GetComparator())),
				"target":     types.StringValue(a.GetTarget()),
			})
			diags.Append(d...)
			objs = append(objs, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: recordAssertionObjTypes}, objs)
		diags.Append(d...)
		data.RecordAssertions = list
	}

	otelObj, otelDiags := openTelemetryFromAPI(ctx, api.GetOpenTelemetry())
	diags.Append(otelDiags...)
	data.OpenTelemetry = otelObj

	return diags
}
