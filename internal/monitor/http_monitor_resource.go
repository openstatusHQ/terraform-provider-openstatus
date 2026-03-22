package monitor

import (
	"context"

	"terraform-provider-openstatus/internal/client"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
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
	Status               types.String `tfsdk:"status"`
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
			},
			"status": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Current monitor status.",
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
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
		},
	}
}

type httpMonitorAPIRequest struct {
	Monitor httpMonitorAPIObject `json:"monitor"`
}

type httpMonitorAPIUpdateRequest struct {
	ID      string               `json:"id"`
	Monitor httpMonitorAPIObject `json:"monitor"`
}

type httpMonitorAPIObject struct {
	ID                   string                       `json:"id,omitempty"`
	Name                 string                       `json:"name"`
	URL                  string                       `json:"url"`
	Periodicity          string                       `json:"periodicity"`
	Method               string                       `json:"method,omitempty"`
	Body                 string                       `json:"body,omitempty"`
	Timeout              jsonInt64                    `json:"timeout,omitempty"`
	DegradedAt           jsonInt64                    `json:"degradedAt,omitempty"`
	Retry                jsonInt64                    `json:"retry,omitempty"`
	FollowRedirects      bool                         `json:"followRedirects,omitempty"`
	Active               bool                         `json:"active,omitempty"`
	Public               bool                         `json:"public,omitempty"`
	Description          string                       `json:"description,omitempty"`
	Regions              []string                     `json:"regions,omitempty"`
	Headers              []apiHeader                  `json:"headers,omitempty"`
	StatusCodeAssertions []apiStatusCodeAssertion     `json:"statusCodeAssertions,omitempty"`
	BodyAssertions       []apiBodyAssertion           `json:"bodyAssertions,omitempty"`
	HeaderAssertions     []apiHeaderAssertion         `json:"headerAssertions,omitempty"`
	Status               string                       `json:"status,omitempty"`
}

type apiHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type apiStatusCodeAssertion struct {
	Target     jsonInt64 `json:"target"`
	Comparator string    `json:"comparator"`
}

type apiBodyAssertion struct {
	Target     string `json:"target"`
	Comparator string `json:"comparator"`
}

type apiHeaderAssertion struct {
	Target     string `json:"target"`
	Comparator string `json:"comparator"`
	Key        string `json:"key"`
}

type httpMonitorAPIResponse struct {
	Monitor httpMonitorAPIObject `json:"monitor"`
}

type getMonitorResponse struct {
	HTTP *httpMonitorAPIObject `json:"http,omitempty"`
	TCP  *tcpMonitorAPIObject  `json:"tcp,omitempty"`
	DNS  *dnsMonitorAPIObject  `json:"dns,omitempty"`
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

	var apiResp httpMonitorAPIResponse
	err := r.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/CreateHTTPMonitor", httpMonitorAPIRequest{Monitor: apiObj}, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error creating HTTP monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(httpAPIToModel(ctx, apiResp.Monitor, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpMonitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data httpMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var apiResp getMonitorResponse
	err := r.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/GetMonitor", map[string]string{"id": data.ID.ValueString()}, &apiResp)
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading HTTP monitor", err.Error())
		return
	}

	if apiResp.HTTP == nil {
		resp.State.RemoveResource(ctx)
		return
	}

	resp.Diagnostics.Append(httpAPIToModel(ctx, *apiResp.HTTP, &data)...)
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

	var apiResp httpMonitorAPIResponse
	err := r.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/UpdateHTTPMonitor",
		httpMonitorAPIUpdateRequest{ID: state.ID.ValueString(), Monitor: apiObj}, &apiResp)
	if err != nil {
		resp.Diagnostics.AddError("Error updating HTTP monitor", err.Error())
		return
	}

	resp.Diagnostics.Append(httpAPIToModel(ctx, apiResp.Monitor, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *httpMonitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data httpMonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Do(ctx, "/openstatus.monitor.v1.MonitorService/DeleteMonitor",
		map[string]string{"id": data.ID.ValueString()}, nil)
	if err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Error deleting HTTP monitor", err.Error())
	}
}

func (r *httpMonitorResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath, types.StringValue(req.ID))...)
}

func httpModelToAPI(ctx context.Context, data httpMonitorModel) (httpMonitorAPIObject, diag.Diagnostics) {
	var diags diag.Diagnostics

	periodicity, err := MapPeriodicityToAPI(data.Periodicity.ValueString())
	if err != nil {
		diags.AddError("Invalid periodicity", err.Error())
		return httpMonitorAPIObject{}, diags
	}

	method, err := MapMethodToAPI(data.Method.ValueString())
	if err != nil {
		diags.AddError("Invalid method", err.Error())
		return httpMonitorAPIObject{}, diags
	}

	var regions []string
	if !data.Regions.IsNull() && !data.Regions.IsUnknown() {
		var tfRegions []string
		diags.Append(data.Regions.ElementsAs(ctx, &tfRegions, false)...)
		if diags.HasError() {
			return httpMonitorAPIObject{}, diags
		}
		regions, err = MapRegionsToAPI(tfRegions)
		if err != nil {
			diags.AddError("Invalid region", err.Error())
			return httpMonitorAPIObject{}, diags
		}
	}

	var headers []apiHeader
	if !data.Headers.IsNull() && !data.Headers.IsUnknown() {
		var tfHeaders []struct {
			Key   string `tfsdk:"key"`
			Value string `tfsdk:"value"`
		}
		diags.Append(data.Headers.ElementsAs(ctx, &tfHeaders, false)...)
		for _, h := range tfHeaders {
			headers = append(headers, apiHeader{Key: h.Key, Value: h.Value})
		}
	}

	var statusCodeAssertions []apiStatusCodeAssertion
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
				return httpMonitorAPIObject{}, diags
			}
			statusCodeAssertions = append(statusCodeAssertions, apiStatusCodeAssertion{
				Target: jsonInt64(a.Target), Comparator: comp,
			})
		}
	}

	var bodyAssertions []apiBodyAssertion
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
				return httpMonitorAPIObject{}, diags
			}
			bodyAssertions = append(bodyAssertions, apiBodyAssertion{
				Target: a.Target, Comparator: comp,
			})
		}
	}

	var headerAssertions []apiHeaderAssertion
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
				return httpMonitorAPIObject{}, diags
			}
			headerAssertions = append(headerAssertions, apiHeaderAssertion{
				Target: a.Target, Comparator: comp, Key: a.Key,
			})
		}
	}

	return httpMonitorAPIObject{
		Name:                 data.Name.ValueString(),
		URL:                  data.URL.ValueString(),
		Periodicity:          periodicity,
		Method:               method,
		Body:                 data.Body.ValueString(),
		Timeout:              jsonInt64(data.Timeout.ValueInt64()),
		DegradedAt:           jsonInt64(data.DegradedAt.ValueInt64()),
		Retry:                jsonInt64(data.Retry.ValueInt64()),
		FollowRedirects:      data.FollowRedirects.ValueBool(),
		Active:               data.Active.ValueBool(),
		Public:               data.Public.ValueBool(),
		Description:          data.Description.ValueString(),
		Regions:              regions,
		Headers:              headers,
		StatusCodeAssertions: statusCodeAssertions,
		BodyAssertions:       bodyAssertions,
		HeaderAssertions:     headerAssertions,
	}, diags
}

func httpAPIToModel(ctx context.Context, api httpMonitorAPIObject, data *httpMonitorModel) diag.Diagnostics {
	var diags diag.Diagnostics

	data.ID = types.StringValue(api.ID)
	data.Name = types.StringValue(api.Name)
	data.URL = types.StringValue(api.URL)
	data.Periodicity = types.StringValue(MapPeriodicityFromAPI(api.Periodicity))
	data.Method = types.StringValue(MapMethodFromAPI(api.Method))
	data.Body = types.StringValue(api.Body)
	data.Timeout = types.Int64Value(api.Timeout.Int64())
	data.DegradedAt = types.Int64Value(api.DegradedAt.Int64())
	data.Retry = types.Int64Value(api.Retry.Int64())
	data.FollowRedirects = types.BoolValue(api.FollowRedirects)
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

	if len(api.Headers) > 0 {
		headerObjs := make([]attr.Value, 0, len(api.Headers))
		for _, h := range api.Headers {
			obj, d := types.ObjectValue(headerObjTypes, map[string]attr.Value{
				"key":   types.StringValue(h.Key),
				"value": types.StringValue(h.Value),
			})
			diags.Append(d...)
			headerObjs = append(headerObjs, obj)
		}
		headerList, d := types.ListValue(types.ObjectType{AttrTypes: headerObjTypes}, headerObjs)
		diags.Append(d...)
		data.Headers = headerList
	}

	if len(api.StatusCodeAssertions) > 0 {
		objs := make([]attr.Value, 0, len(api.StatusCodeAssertions))
		for _, a := range api.StatusCodeAssertions {
			obj, d := types.ObjectValue(statusCodeAssertionObjTypes, map[string]attr.Value{
				"target":     types.Int64Value(a.Target.Int64()),
				"comparator": types.StringValue(MapNumberComparatorFromAPI(a.Comparator)),
			})
			diags.Append(d...)
			objs = append(objs, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: statusCodeAssertionObjTypes}, objs)
		diags.Append(d...)
		data.StatusCodeAssertions = list
	}

	if len(api.BodyAssertions) > 0 {
		objs := make([]attr.Value, 0, len(api.BodyAssertions))
		for _, a := range api.BodyAssertions {
			obj, d := types.ObjectValue(bodyAssertionObjTypes, map[string]attr.Value{
				"target":     types.StringValue(a.Target),
				"comparator": types.StringValue(MapStringComparatorFromAPI(a.Comparator)),
			})
			diags.Append(d...)
			objs = append(objs, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: bodyAssertionObjTypes}, objs)
		diags.Append(d...)
		data.BodyAssertions = list
	}

	if len(api.HeaderAssertions) > 0 {
		objs := make([]attr.Value, 0, len(api.HeaderAssertions))
		for _, a := range api.HeaderAssertions {
			obj, d := types.ObjectValue(headerAssertionObjTypes, map[string]attr.Value{
				"target":     types.StringValue(a.Target),
				"comparator": types.StringValue(MapStringComparatorFromAPI(a.Comparator)),
				"key":        types.StringValue(a.Key),
			})
			diags.Append(d...)
			objs = append(objs, obj)
		}
		list, d := types.ListValue(types.ObjectType{AttrTypes: headerAssertionObjTypes}, objs)
		diags.Append(d...)
		data.HeaderAssertions = list
	}

	return diags
}

func isNotFound(err error) bool {
	if apiErr, ok := err.(*client.APIError); ok {
		return apiErr.Code == "not_found"
	}
	return false
}
