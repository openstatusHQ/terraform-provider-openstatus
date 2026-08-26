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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*httpMonitorResource)(nil)
	_ resource.ResourceWithConfigure   = (*httpMonitorResource)(nil)
	_ resource.ResourceWithImportState = (*httpMonitorResource)(nil)
)

func NewHTTPMonitorResource() resource.Resource {
	return &httpMonitorResource{}
}

type httpMonitorResource struct {
	client *client.Client
}

type httpMonitorModel struct {
	ID                   types.String `tfsdk:"id"`
	Name                 types.String `tfsdk:"name"`
	URL                  types.String `tfsdk:"url"`
	Periodicity          types.String `tfsdk:"periodicity"`
	Method               types.String `tfsdk:"method"`
	Body                 types.String `tfsdk:"body"`
	Timeout              types.Int64  `tfsdk:"timeout"`
	DegradedAt           types.Int64  `tfsdk:"degraded_at"`
	Retry                types.Int64  `tfsdk:"retry"`
	FollowRedirects      types.Bool   `tfsdk:"follow_redirects"`
	Active               types.Bool   `tfsdk:"active"`
	Public               types.Bool   `tfsdk:"public"`
	Description          types.String `tfsdk:"description"`
	Regions              types.Set    `tfsdk:"regions"`
	Headers              types.List   `tfsdk:"headers"`
	StatusCodeAssertions types.List   `tfsdk:"status_code_assertions"`
	BodyAssertions       types.List   `tfsdk:"body_assertions"`
	HeaderAssertions     types.List   `tfsdk:"header_assertions"`
	OpenTelemetry        types.Object `tfsdk:"open_telemetry"`
	Status               types.String `tfsdk:"status"`
	PrivateLocationIDs   types.Set    `tfsdk:"private_location_ids"`
}

func (r *httpMonitorResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_http_monitor"
}

func (r *httpMonitorResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	r.client = req.ProviderData.(providerConfig).Client
}

var headerObjTypes = map[string]attr.Type{
	"key":   types.StringType,
	"value": types.StringType,
}

var statusCodeAssertionObjTypes = map[string]attr.Type{
	"target":     types.Int64Type,
	"comparator": types.StringType,
}

var bodyAssertionObjTypes = map[string]attr.Type{
	"target":     types.StringType,
	"comparator": types.StringType,
}

var headerAssertionObjTypes = map[string]attr.Type{
	"target":     types.StringType,
	"comparator": types.StringType,
	"key":        types.StringType,
}

func (r *httpMonitorResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an HTTP monitor.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Monitor identifier.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Monitor name.",
				Validators:          []validator.String{stringvalidator.LengthAtMost(256)},
			},
			"url": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "URL to monitor.",
				Validators:          []validator.String{stringvalidator.LengthAtMost(2048)},
			},
			"periodicity": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "How often the monitor runs.",
				Validators:          []validator.String{stringvalidator.OneOf(PeriodicityValues...)},
			},
			"method": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("GET"),
				MarkdownDescription: "HTTP method.",
				Validators:          []validator.String{stringvalidator.OneOf(MethodValues...)},
			},
			"body": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(""),
				MarkdownDescription: "Request body.",
			},
			"timeout": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(45000),
				MarkdownDescription: "Timeout in milliseconds (0–120000).",
				Validators:          []validator.Int64{int64validator.Between(0, 120000)},
			},
			"degraded_at": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Degradation threshold in milliseconds (0–120000).",
				Validators:          []validator.Int64{int64validator.Between(0, 120000)},
			},
			"retry": schema.Int64Attribute{
				Optional:            true,
				Computed:            true,
				Default:             int64default.StaticInt64(3),
				MarkdownDescription: "Number of retries (0–10).",
				Validators:          []validator.Int64{int64validator.Between(0, 10)},
			},
			"follow_redirects": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
				MarkdownDescription: "Whether to follow redirects.",
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the monitor is active.",
			},
			"public": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the monitor is publicly visible.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Monitor description.",
				Validators:          []validator.String{stringvalidator.LengthAtMost(1024)},
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
				Computed:            true,
				MarkdownDescription: "Current monitor status.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"private_location_ids": schema.SetAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "IDs of the private locations that run this monitor. Managed from `openstatus_private_location.monitor_ids`.",
			},
		},
		Blocks: map[string]schema.Block{
			"headers": schema.ListNestedBlock{
				MarkdownDescription: "Custom HTTP headers.",
				Validators:          []validator.List{listvalidator.SizeAtMost(20)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"key":   schema.StringAttribute{Required: true},
						"value": schema.StringAttribute{Required: true},
					},
				},
			},
			"status_code_assertions": schema.ListNestedBlock{
				MarkdownDescription: "Status code assertions.",
				Validators:          []validator.List{listvalidator.SizeAtMost(10)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"target": schema.Int64Attribute{
							Required:   true,
							Validators: []validator.Int64{int64validator.Between(100, 599)},
						},
						"comparator": schema.StringAttribute{
							Required:   true,
							Validators: []validator.String{stringvalidator.OneOf(NumberComparatorValues...)},
						},
					},
				},
			},
			"body_assertions": schema.ListNestedBlock{
				MarkdownDescription: "Body assertions.",
				Validators:          []validator.List{listvalidator.SizeAtMost(10)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"target": schema.StringAttribute{Required: true},
						"comparator": schema.StringAttribute{
							Required:   true,
							Validators: []validator.String{stringvalidator.OneOf(StringComparatorValues...)},
						},
					},
				},
			},
			"header_assertions": schema.ListNestedBlock{
				MarkdownDescription: "Header assertions.",
				Validators:          []validator.List{listvalidator.SizeAtMost(10)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"target": schema.StringAttribute{Required: true},
						"comparator": schema.StringAttribute{
							Required:   true,
							Validators: []validator.String{stringvalidator.OneOf(StringComparatorValues...)},
						},
						"key": schema.StringAttribute{
							Required:   true,
							Validators: []validator.String{stringvalidator.LengthAtLeast(1)},
						},
					},
				},
			},
			"open_telemetry": openTelemetrySchemaBlock(),
		},
	}
}

func (r *httpMonitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data httpMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := httpModelToAPI(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiReq := &monitorv1.CreateHTTPMonitorRequest{}
	apiReq.SetMonitor(apiObj)

	apiResp, err := r.client.Monitor.CreateHTTPMonitor(ctx, connect.NewRequest(apiReq))
	if err != nil {
		resp.Diagnostics.AddError("Error creating HTTP monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(httpAPIToModel(ctx, apiResp.Msg.GetMonitor(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data httpMonitorModel
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
		resp.Diagnostics.AddError("Error reading HTTP monitor", err.Error())
		return
	}

	monitor := apiResp.Msg.GetMonitor().GetHttp()
	if monitor == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(httpAPIToModel(ctx, monitor, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpMonitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data httpMonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var state httpMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiObj, diags := httpModelToAPI(ctx, data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	updateReq := &monitorv1.UpdateHTTPMonitorRequest{}
	updateReq.SetId(state.ID.ValueString())
	updateReq.SetMonitor(apiObj)

	apiResp, err := r.client.Monitor.UpdateHTTPMonitor(ctx, connect.NewRequest(updateReq))
	if err != nil {
		resp.Diagnostics.AddError("Error updating HTTP monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(httpAPIToModel(ctx, apiResp.Msg.GetMonitor(), &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data httpMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := &monitorv1.DeleteMonitorRequest{}
	deleteReq.SetId(data.ID.ValueString())

	_, err := r.client.Monitor.DeleteMonitor(ctx, connect.NewRequest(deleteReq))
	if err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting HTTP monitor", err.Error())
	}
}

func (r *httpMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath, types.StringValue(req.ID))...)
}

func httpModelToAPI(ctx context.Context, data httpMonitorModel) (*monitorv1.HTTPMonitor, diag.Diagnostics) {
	var diags diag.Diagnostics

	periodicity, err := MapPeriodicityToAPI(data.Periodicity.ValueString())
	if err != nil {
		diags.AddError("Invalid periodicity", err.Error())
		return nil, diags
	}

	method, err := MapMethodToAPI(data.Method.ValueString())
	if err != nil {
		diags.AddError("Invalid method", err.Error())
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

	var headers []*monitorv1.Headers
	if !data.Headers.IsNull() && !data.Headers.IsUnknown() {
		var tfHeaders []struct {
			Key   string `tfsdk:"key"`
			Value string `tfsdk:"value"`
		}
		diags.Append(data.Headers.ElementsAs(ctx, &tfHeaders, false)...)
		for _, h := range tfHeaders {
			header := &monitorv1.Headers{}
			header.SetKey(h.Key)
			header.SetValue(h.Value)
			headers = append(headers, header)
		}
	}

	var statusCodeAssertions []*monitorv1.StatusCodeAssertion
	if !data.StatusCodeAssertions.IsNull() && !data.StatusCodeAssertions.IsUnknown() {
		var tfAssertions []struct {
			Target     int64  `tfsdk:"target"`
			Comparator string `tfsdk:"comparator"`
		}
		diags.Append(data.StatusCodeAssertions.ElementsAs(ctx, &tfAssertions, false)...)
		for _, a := range tfAssertions {
			comp, mapErr := MapNumberComparatorToAPI(a.Comparator)
			if mapErr != nil {
				diags.AddError("Invalid comparator", mapErr.Error())
				return nil, diags
			}
			assertion := &monitorv1.StatusCodeAssertion{}
			assertion.SetTarget(a.Target)
			assertion.SetComparator(comp)
			statusCodeAssertions = append(statusCodeAssertions, assertion)
		}
	}

	var bodyAssertions []*monitorv1.BodyAssertion
	if !data.BodyAssertions.IsNull() && !data.BodyAssertions.IsUnknown() {
		var tfAssertions []struct {
			Target     string `tfsdk:"target"`
			Comparator string `tfsdk:"comparator"`
		}
		diags.Append(data.BodyAssertions.ElementsAs(ctx, &tfAssertions, false)...)
		for _, a := range tfAssertions {
			comp, mapErr := MapStringComparatorToAPI(a.Comparator)
			if mapErr != nil {
				diags.AddError("Invalid comparator", mapErr.Error())
				return nil, diags
			}
			assertion := &monitorv1.BodyAssertion{}
			assertion.SetTarget(a.Target)
			assertion.SetComparator(comp)
			bodyAssertions = append(bodyAssertions, assertion)
		}
	}

	var headerAssertions []*monitorv1.HeaderAssertion
	if !data.HeaderAssertions.IsNull() && !data.HeaderAssertions.IsUnknown() {
		var tfAssertions []struct {
			Target     string `tfsdk:"target"`
			Comparator string `tfsdk:"comparator"`
			Key        string `tfsdk:"key"`
		}
		diags.Append(data.HeaderAssertions.ElementsAs(ctx, &tfAssertions, false)...)
		for _, a := range tfAssertions {
			comp, mapErr := MapStringComparatorToAPI(a.Comparator)
			if mapErr != nil {
				diags.AddError("Invalid comparator", mapErr.Error())
				return nil, diags
			}
			assertion := &monitorv1.HeaderAssertion{}
			assertion.SetTarget(a.Target)
			assertion.SetComparator(comp)
			assertion.SetKey(a.Key)
			headerAssertions = append(headerAssertions, assertion)
		}
	}

	otel, otelDiags := openTelemetryToAPI(ctx, data.OpenTelemetry)
	diags.Append(otelDiags...)
	if diags.HasError() {
		return nil, diags
	}

	out := &monitorv1.HTTPMonitor{}
	out.SetName(data.Name.ValueString())
	out.SetUrl(data.URL.ValueString())
	out.SetPeriodicity(periodicity)
	out.SetMethod(method)
	out.SetBody(data.Body.ValueString())
	out.SetTimeout(data.Timeout.ValueInt64())
	out.SetRetry(data.Retry.ValueInt64())
	out.SetFollowRedirects(data.FollowRedirects.ValueBool())
	out.SetActive(data.Active.ValueBool())
	out.SetPublic(data.Public.ValueBool())
	out.SetDescription(data.Description.ValueString())
	out.SetRegions(regions)
	out.SetHeaders(headers)
	out.SetStatusCodeAssertions(statusCodeAssertions)
	out.SetBodyAssertions(bodyAssertions)
	out.SetHeaderAssertions(headerAssertions)
	if otel != nil {
		out.SetOpenTelemetry(otel)
	}
	// degraded_at is proto-optional and was previously omitted when zero.
	if v := data.DegradedAt.ValueInt64(); v != 0 {
		out.SetDegradedAt(v)
	}
	return out, diags
}

func httpAPIToModel(ctx context.Context, api *monitorv1.HTTPMonitor, data *httpMonitorModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(api.GetId())
	data.Name = types.StringValue(api.GetName())
	data.URL = types.StringValue(api.GetUrl())
	data.Periodicity = types.StringValue(MapPeriodicityFromAPI(api.GetPeriodicity()))
	data.Method = types.StringValue(MapMethodFromAPI(api.GetMethod()))
	data.Body = types.StringValue(api.GetBody())
	data.Timeout = types.Int64Value(api.GetTimeout())
	data.DegradedAt = types.Int64Value(api.GetDegradedAt())
	data.Retry = types.Int64Value(api.GetRetry())
	data.FollowRedirects = types.BoolValue(api.GetFollowRedirects())
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

	// API returns `headers: [{}]` as a placeholder when no headers are
	// configured. Drop empty entries so block count matches the plan (#19).
	headerObjs := make([]attr.Value, 0, len(api.GetHeaders()))
	for _, h := range api.GetHeaders() {
		if h.GetKey() == "" && h.GetValue() == "" {
			continue
		}
		obj, d := types.ObjectValue(headerObjTypes, map[string]attr.Value{
			"key":   types.StringValue(h.GetKey()),
			"value": types.StringValue(h.GetValue()),
		})
		diags.Append(d...)
		headerObjs = append(headerObjs, obj)
	}
	headerList, d := types.ListValue(types.ObjectType{AttrTypes: headerObjTypes}, headerObjs)
	diags.Append(d...)
	data.Headers = headerList

	if len(api.GetStatusCodeAssertions()) > 0 {
		objs := make([]attr.Value, 0, len(api.GetStatusCodeAssertions()))
		for _, a := range api.GetStatusCodeAssertions() {
			obj, d := types.ObjectValue(statusCodeAssertionObjTypes, map[string]attr.Value{
				"target":     types.Int64Value(a.GetTarget()),
				"comparator": types.StringValue(MapNumberComparatorFromAPI(a.GetComparator())),
			})
			diags.Append(d...)
			objs = append(objs, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: statusCodeAssertionObjTypes}, objs)
		diags.Append(d...)
		data.StatusCodeAssertions = list
	}

	if len(api.GetBodyAssertions()) > 0 {
		objs := make([]attr.Value, 0, len(api.GetBodyAssertions()))
		for _, a := range api.GetBodyAssertions() {
			obj, d := types.ObjectValue(bodyAssertionObjTypes, map[string]attr.Value{
				"target":     types.StringValue(a.GetTarget()),
				"comparator": types.StringValue(MapStringComparatorFromAPI(a.GetComparator())),
			})
			diags.Append(d...)
			objs = append(objs, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: bodyAssertionObjTypes}, objs)
		diags.Append(d...)
		data.BodyAssertions = list
	}

	if len(api.GetHeaderAssertions()) > 0 {
		objs := make([]attr.Value, 0, len(api.GetHeaderAssertions()))
		for _, a := range api.GetHeaderAssertions() {
			obj, d := types.ObjectValue(headerAssertionObjTypes, map[string]attr.Value{
				"target":     types.StringValue(a.GetTarget()),
				"comparator": types.StringValue(MapStringComparatorFromAPI(a.GetComparator())),
				"key":        types.StringValue(a.GetKey()),
			})
			diags.Append(d...)
			objs = append(objs, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: headerAssertionObjTypes}, objs)
		diags.Append(d...)
		data.HeaderAssertions = list
	}

	otelObj, otelDiags := openTelemetryFromAPI(ctx, api.GetOpenTelemetry(), data.OpenTelemetry)
	diags.Append(otelDiags...)
	data.OpenTelemetry = otelObj

	return diags
}
