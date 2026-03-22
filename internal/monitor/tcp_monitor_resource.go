package monitor

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	URI         types.String `tfsdk:"uri"`
	Periodicity types.String `tfsdk:"periodicity"`
	Timeout     types.Int64  `tfsdk:"timeout"`
	DegradedAt  types.Int64  `tfsdk:"degraded_at"`
	Retry       types.Int64  `tfsdk:"retry"`
	Active      types.Bool   `tfsdk:"active"`
	Public      types.Bool   `tfsdk:"public"`
	Description types.String `tfsdk:"description"`
	Regions     types.Set    `tfsdk:"regions"`
	Status      types.String `tfsdk:"status"`
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
				Optional:    true,
				ElementType: types.StringType,
			},
			"status": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

type tcpMonitorAPIObject struct {
	ID          string    `json:"id,omitempty"`
	Name        string    `json:"name"`
	URI         string    `json:"uri"`
	Periodicity string    `json:"periodicity"`
	Timeout     jsonInt64 `json:"timeout,omitempty"`
	DegradedAt  jsonInt64 `json:"degradedAt,omitempty"`
	Retry       jsonInt64 `json:"retry,omitempty"`
	Active      bool     `json:"active,omitempty"`
	Public      bool     `json:"public,omitempty"`
	Description string   `json:"description,omitempty"`
	Regions     []string `json:"regions,omitempty"`
	Status      string   `json:"status,omitempty"`
}

type tcpMonitorAPIRequest struct {
	Monitor tcpMonitorAPIObject `json:"monitor"`
}

type tcpMonitorAPIUpdateRequest struct {
	ID      string              `json:"id"`
	Monitor tcpMonitorAPIObject `json:"monitor"`
}

type tcpMonitorAPIResponse struct {
	Monitor tcpMonitorAPIObject `json:"monitor"`
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

	var apiResp tcpMonitorAPIResponse
	err := r.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/CreateTCPMonitor",
		tcpMonitorAPIRequest{Monitor: apiObj}, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error creating TCP monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(tcpAPIToModel(ctx, apiResp.Monitor, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tcpMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data tcpMonitorModel
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
		resp.Diagnostics.AddError("Error reading TCP monitor", err.Error())
		return
	}

	if apiResp.TCP == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(tcpAPIToModel(ctx, *apiResp.TCP, &data)...)
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

	var apiResp tcpMonitorAPIResponse
	err := r.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/UpdateTCPMonitor",
		tcpMonitorAPIUpdateRequest{ID: state.ID.ValueString(), Monitor: apiObj}, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error updating TCP monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(tcpAPIToModel(ctx, apiResp.Monitor, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *tcpMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data tcpMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/DeleteMonitor",
		map[string]string{"id": data.ID.ValueString()}, nil)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting TCP monitor", err.Error())
	}
}

func (r *tcpMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath, types.StringValue(req.ID))...)
}

func tcpModelToAPI(ctx context.Context, data tcpMonitorModel) (tcpMonitorAPIObject, diag.Diagnostics) {
	var diags diag.Diagnostics

	periodicity, err := MapPeriodicityToAPI(data.Periodicity.ValueString())
	if err != nil {
		diags.AddError("Invalid periodicity", err.Error())
		return tcpMonitorAPIObject{}, diags
	}

	var regions []string
	if !data.Regions.IsNull() && !data.Regions.IsUnknown() {
		var tfRegions []string
		diags.Append(data.Regions.ElementsAs(ctx, &tfRegions, false)...)
		if diags.HasError() {
			return tcpMonitorAPIObject{}, diags
		}
		regions, err = MapRegionsToAPI(tfRegions)
		if err != nil {
			diags.AddError("Invalid region", err.Error())
			return tcpMonitorAPIObject{}, diags
		}
	}

	return tcpMonitorAPIObject{
		Name:        data.Name.ValueString(),
		URI:         data.URI.ValueString(),
		Periodicity: periodicity,
		Timeout:     jsonInt64(data.Timeout.ValueInt64()),
		DegradedAt:  jsonInt64(data.DegradedAt.ValueInt64()),
		Retry:       jsonInt64(data.Retry.ValueInt64()),
		Active:      data.Active.ValueBool(),
		Public:      data.Public.ValueBool(),
		Description: data.Description.ValueString(),
		Regions:     regions,
	}, diags
}

func tcpAPIToModel(ctx context.Context, api tcpMonitorAPIObject, data *tcpMonitorModel) diag.Diagnostics {
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

	return diags
}
