package monitor

import (
	"context"

	"terraform-provider-openstatus/internal/client"

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

type dnsMonitorAPIObject struct {
	ID               string               `json:"id,omitempty"`
	Name             string               `json:"name"`
	URI              string               `json:"uri"`
	Periodicity      string               `json:"periodicity"`
	Timeout          jsonInt64            `json:"timeout"`
	DegradedAt       jsonInt64            `json:"degradedAt,omitempty"`
	Retry            jsonInt64            `json:"retry"`
	Active           bool                 `json:"active"`
	Public           bool                 `json:"public"`
	Description      string               `json:"description"`
	Regions          []string             `json:"regions,omitempty"`
	RecordAssertions []apiRecordAssertion `json:"recordAssertions,omitempty"`
	OpenTelemetry    *apiOpenTelemetry    `json:"openTelemetry"`
	Status           string               `json:"status,omitempty"`
}

type apiRecordAssertion struct {
	Record     string `json:"record"`
	Comparator string `json:"comparator"`
	Target     string `json:"target"`
}

type dnsMonitorAPIRequest struct {
	Monitor dnsMonitorAPIObject `json:"monitor"`
}

type dnsMonitorAPIUpdateRequest struct {
	ID      string              `json:"id"`
	Monitor dnsMonitorAPIObject `json:"monitor"`
}

type dnsMonitorAPIResponse struct {
	Monitor dnsMonitorAPIObject `json:"monitor"`
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

	var apiResp dnsMonitorAPIResponse
	err := r.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/CreateDNSMonitor",
		dnsMonitorAPIRequest{Monitor: apiObj}, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error creating DNS monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(dnsAPIToModel(ctx, apiResp.Monitor, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data dnsMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResp getMonitorResponse
	err := r.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/GetMonitor",
		map[string]string{"id": data.ID.ValueString()}, &apiResp)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading DNS monitor", err.Error())
		return
	}

	if apiResp.Monitor.DNS == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(dnsAPIToModel(ctx, *apiResp.Monitor.DNS, &data)...)
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

	var apiResp dnsMonitorAPIResponse
	err := r.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/UpdateDNSMonitor",
		dnsMonitorAPIUpdateRequest{ID: state.ID.ValueString(), Monitor: apiObj}, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error updating DNS monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(dnsAPIToModel(ctx, apiResp.Monitor, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *dnsMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data dnsMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/DeleteMonitor",
		map[string]string{"id": data.ID.ValueString()}, nil)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting DNS monitor", err.Error())
	}
}

func (r *dnsMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath, types.StringValue(req.ID))...)
}

func dnsModelToAPI(ctx context.Context, data dnsMonitorModel) (dnsMonitorAPIObject, diag.Diagnostics) {
	var diags diag.Diagnostics

	periodicity, err := MapPeriodicityToAPI(data.Periodicity.ValueString())
	if err != nil {
		diags.AddError("Invalid periodicity", err.Error())
		return dnsMonitorAPIObject{}, diags
	}

	var regions []string
	if !data.Regions.IsNull() && !data.Regions.IsUnknown() {
		var tfRegions []string
		diags.Append(data.Regions.ElementsAs(ctx, &tfRegions, false)...)
		if diags.HasError() {
			return dnsMonitorAPIObject{}, diags
		}
		regions, err = MapRegionsToAPI(tfRegions)
		if err != nil {
			diags.AddError("Invalid region", err.Error())
			return dnsMonitorAPIObject{}, diags
		}
	}

	var recordAssertions []apiRecordAssertion
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
				return dnsMonitorAPIObject{}, diags
			}
			recordAssertions = append(recordAssertions, apiRecordAssertion{
				Record: a.Record, Comparator: comp, Target: a.Target,
			})
		}
	}

	otel, otelDiags := openTelemetryToAPI(ctx, data.OpenTelemetry)
	diags.Append(otelDiags...)
	if diags.HasError() {
		return dnsMonitorAPIObject{}, diags
	}

	return dnsMonitorAPIObject{
		Name:             data.Name.ValueString(),
		URI:              data.URI.ValueString(),
		Periodicity:      periodicity,
		Timeout:          jsonInt64(data.Timeout.ValueInt64()),
		DegradedAt:       jsonInt64(data.DegradedAt.ValueInt64()),
		Retry:            jsonInt64(data.Retry.ValueInt64()),
		Active:           data.Active.ValueBool(),
		Public:           data.Public.ValueBool(),
		Description:      data.Description.ValueString(),
		Regions:          regions,
		RecordAssertions: recordAssertions,
		OpenTelemetry:    otel,
	}, diags
}

func dnsAPIToModel(ctx context.Context, api dnsMonitorAPIObject, data *dnsMonitorModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.URI = types.StringValue(api.URI)
	data.Periodicity = types.StringValue(MapPeriodicityFromAPI(api.Periodicity))
	data.Timeout = types.Int64Value(api.Timeout.Int64())
	data.DegradedAt = types.Int64Value(api.DegradedAt.Int64())
	data.Retry = types.Int64Value(api.Retry.Int64())
	data.Active = types.BoolValue(api.Active)
	data.Public = types.BoolValue(api.Public)
	if api.Description != "" || !data.Description.IsNull() {
		data.Description = types.StringValue(api.Description)
	}
	data.Status = types.StringValue(MapMonitorStatusFromAPI(api.Status))

	if len(api.Regions) > 0 {
		regionVals := MapRegionsFromAPI(api.Regions)
		regionSet, d := types.SetValueFrom(ctx, types.StringType, regionVals)
		diags.Append(d...)
		data.Regions = regionSet
	} else if !data.Regions.IsNull() {
		data.Regions = types.SetNull(types.StringType)
	}

	if len(api.RecordAssertions) > 0 {
		objs := make([]attr.Value, 0, len(api.RecordAssertions))
		for _, a := range api.RecordAssertions {
			obj, d := types.ObjectValue(recordAssertionObjTypes, map[string]attr.Value{
				"record":     types.StringValue(a.Record),
				"comparator": types.StringValue(MapRecordComparatorFromAPI(a.Comparator)),
				"target":     types.StringValue(a.Target),
			})
			diags.Append(d...)
			objs = append(objs, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: recordAssertionObjTypes}, objs)
		diags.Append(d...)
		data.RecordAssertions = list
	}

	otelObj, otelDiags := openTelemetryFromAPI(ctx, api.OpenTelemetry)
	diags.Append(otelDiags...)
	data.OpenTelemetry = otelObj

	return diags
}
