package provider

import (
	"context"
	"encoding/json"
	"math/big"
	"strconv"

	"terraform-provider-openstatus/client"
	"terraform-provider-openstatus/internal/resource_monitor"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	hreq "github.com/imroc/req/v3"
)

var _ resource.Resource = (*monitorResource)(nil)

func NewMonitorResource() resource.Resource {
	return &monitorResource{}
}

type monitorResource struct {
	client *hreq.Client
}

func (r *monitorResource) Configure(_ context.Context, req resource.ConfigureRequest, _ *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	config := req.ProviderData.(ProviderConfig)
	r.client = config.client
}

func (r *monitorResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_monitor"
}

func (r *monitorResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = resource_monitor.MonitorResourceSchema(ctx)
}

func (r *monitorResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data resource_monitor.MonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(bindObject(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var regions []string
	regionsTF := make([]types.String, 0, len(data.Regions.Elements()))
	diags := data.Regions.ElementsAs(ctx, &regions, false)
	if diags.HasError() {
		return
	}

	for _, region := range regionsTF {
		regions = append(regions, region.ValueString())
	}

	var headers []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	var headersTF []resource_monitor.HeadersValue
	diags = data.Headers.ElementsAs(ctx, &headersTF, true)
	if diags.HasError() {
		return
	}
	for _, header := range headersTF {
		headers = append(headers, struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}{
			Key:   header.Key.ValueString(),
			Value: header.Value.ValueString(),
		})
	}

	var assertions []interface{}

	var assertionsTF []resource_monitor.AssertionsValue
	diags = data.Assertions.ElementsAs(ctx, &assertionsTF, true)
	if diags.HasError() {
		return
	}

	for _, assert := range assertionsTF {
		if assert.AssertionsType.ValueString() == "status" {
			i, _ := strconv.Atoi(assert.Target.ValueString())

			assertions = append(assertions, struct {
				Target  int    `json:"target"`
				Type    string `json:"type"`
				Compare string `json:"compare"`
			}{
				Target:  i,
				Type:    assert.AssertionsType.ValueString(),
				Compare: assert.Compare.ValueString(),
			})
		} else {
			assertions = append(assertions, struct {
				Target  string `json:"target"`
				Type    string `json:"type"`
				Compare string `json:"compare"`
				Key     string `json:"key"`
			}{
				Target:  assert.Target.ValueString(),
				Type:    assert.AssertionsType.ValueString(),
				Compare: assert.Compare.ValueString(),
				Key:     assert.Key.ValueString(),
			})
		}

	}
	timeout, _ := data.Timeout.ValueBigFloat().Int64()
	degradedAfter, _ := data.DegradedAfter.ValueBigFloat().Int64()
	b, _ := json.Marshal(assertions)

	t := json.RawMessage(b)

	out, err := client.CreateMonitor(ctx, r.client, client.MonitorRequest{
		Active:      data.Active.ValueBool(),
		Body:        data.Body.ValueString(),
		Description: data.Description.ValueString(),
		Headers:     headers,

		Url:           data.Url.ValueString(),
		Name:          data.Name.ValueString(),
		Periodicity:   data.Periodicity.ValueString(),
		Regions:       regions,
		Method:        data.Method.ValueString(),
		Timeout:       int(timeout),
		DegradedAfter: int(degradedAfter),
		Assertions:    t,
	})

	if err != nil {
		resp.Diagnostics.AddError("Error creating monitor", "Could not create the monitor:"+err.Error())
		return
	}

	data.Id = types.NumberValue(big.NewFloat(float64(out.Id)))
	data.Active = types.BoolValue(out.Active)
	data.Body = types.StringValue(out.Body)
	data.Description = types.StringValue(out.Description)
	data.Url = types.StringValue(out.Url)
	data.Name = types.StringValue(out.Name)
	data.Periodicity = types.StringValue(out.Periodicity)
	data.Method = types.StringValue(out.Method)
	data.DegradedAfter = types.NumberValue(big.NewFloat(float64(out.DegradedAfter)))
	data.Timeout = types.NumberValue(big.NewFloat(float64(out.Timeout)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *monitorResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data resource_monitor.MonitorModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !data.Id.IsNull() {
		monitor, err := client.GetMonitor(ctx, r.client, data.Id.String())
		if err != nil {
			resp.Diagnostics.AddError("Error reading monitor", "Could not read the monitor:"+err.Error())
			return
		}
		data.Id = types.NumberValue(big.NewFloat(float64(monitor.Id)))
		data.Active = types.BoolValue(monitor.Active)
		data.Body = types.StringValue(monitor.Body)
		data.Description = types.StringValue(monitor.Description)
		data.Url = types.StringValue(monitor.Url)
		data.Name = types.StringValue(monitor.Name)
		data.Periodicity = types.StringValue(monitor.Periodicity)
		data.Method = types.StringValue(monitor.Method)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

}

func (r *monitorResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {

	var before resource_monitor.MonitorModel
	req.State.Get(ctx, &before)

	var data resource_monitor.MonitorModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(bindObject(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var regions []string
	regionsTF := make([]types.String, 0, len(data.Regions.Elements()))
	diags := data.Regions.ElementsAs(ctx, &regions, false)
	if diags.HasError() {
		return
	}

	for _, region := range regionsTF {
		regions = append(regions, region.ValueString())
	}

	var headers []struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}

	var headersTF []resource_monitor.HeadersValue
	diags = data.Headers.ElementsAs(ctx, &headersTF, true)
	if diags.HasError() {
		return
	}
	for _, header := range headersTF {
		headers = append(headers, struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		}{
			Key:   header.Key.ValueString(),
			Value: header.Value.ValueString(),
		})
	}

	var assertions []interface{}

	var assertionsTF []resource_monitor.AssertionsValue
	diags = data.Assertions.ElementsAs(ctx, &assertionsTF, true)
	if diags.HasError() {
		return
	}
	for _, assert := range assertionsTF {
		if assert.AssertionsType.ValueString() == "status" {

			i, _ := strconv.Atoi(assert.Target.ValueString())
			assertions = append(assertions, struct {
				Target  int    `json:"target"`
				Type    string `json:"type"`
				Compare string `json:"compare"`
			}{
				Target:  i,
				Type:    assert.AssertionsType.ValueString(),
				Compare: assert.Compare.ValueString(),
			})
		} else {
			assertions = append(assertions, struct {
				Target  string `json:"target"`
				Type    string `json:"type"`
				Compare string `json:"compare"`
				Key     string `json:"key"`
			}{
				Target:  assert.Target.ValueString(),
				Type:    assert.AssertionsType.ValueString(),
				Compare: assert.Compare.ValueString(),
				Key:     assert.Key.ValueString(),
			})
		}
	}

	timeout, _ := data.Timeout.ValueBigFloat().Int64()
	degradedAfter, _ := data.DegradedAfter.ValueBigFloat().Int64()

	b, _ := json.Marshal(assertions)

	t := json.RawMessage(b)
	json.RawMessage.MarshalJSON(b)
	out, err := client.UpdateMonitor(ctx, r.client, client.MonitorRequest{
		Active:      data.Active.ValueBool(),
		Body:        data.Body.ValueString(),
		Description: data.Description.ValueString(),
		Headers:     headers,

		Url:           data.Url.ValueString(),
		Name:          data.Name.ValueString(),
		Periodicity:   data.Periodicity.ValueString(),
		Regions:       regions,
		Method:        data.Method.ValueString(),
		Timeout:       int(timeout),
		DegradedAfter: int(degradedAfter),
		Assertions:    t,
	}, data.Id.String())

	if err != nil {
		resp.Diagnostics.AddError("Error creating monitor", "Could not create the monitor:")
		return
	}

	data.Id = types.NumberValue(big.NewFloat(float64(out.Id)))
	data.Active = types.BoolValue(out.Active)
	data.Body = types.StringValue(out.Body)
	data.Description = types.StringValue(out.Description)
	data.Url = types.StringValue(out.Url)
	data.Name = types.StringValue(out.Name)
	data.Periodicity = types.StringValue(out.Periodicity)
	data.Method = types.StringValue(out.Method)
	data.DegradedAfter = types.NumberValue(big.NewFloat(float64(out.DegradedAfter)))
	data.Timeout = types.NumberValue(big.NewFloat(float64(out.Timeout)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

}

func (r *monitorResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data resource_monitor.MonitorModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := client.DeleteMonitor(ctx, r.client, data.Id.String())
	if err != nil {
		resp.Diagnostics.AddError("Error creating monitor", "Could not create the monitor:")
		return
	}
}

func bindObject(ctx context.Context, monitor *resource_monitor.MonitorModel) diag.Diagnostics {

	if monitor.Id.IsUnknown() {
		monitor.Id = types.NumberNull()
	}
	if monitor.Name.IsUnknown() {
		monitor.Name = types.StringNull()
	}
	if monitor.Url.IsUnknown() {
		monitor.Url = types.StringNull()
	}

	if monitor.Method.IsUnknown() {
		monitor.Method = types.StringNull()
	}
	if monitor.Body.IsUnknown() {
		monitor.Body = types.StringNull()
	}
	if monitor.Description.IsUnknown() {
		monitor.Description = types.StringNull()
	}
	if monitor.Timeout.IsUnknown() {
		monitor.Timeout = types.NumberNull()
	}
	if monitor.DegradedAfter.IsUnknown() {
		monitor.DegradedAfter = types.NumberNull()
	}

	if monitor.Regions.IsNull() {
		monitor.Regions = types.ListNull(types.StringNull().Type(ctx))
	} else if !monitor.Regions.IsNull() {
		var regions []string
		diags := monitor.Regions.ElementsAs(ctx, &regions, false)
		if diags.HasError() {
			return diags
		}

		monitor.Regions, diags = types.ListValueFrom(ctx, types.StringUnknown().Type(ctx), regions)
		if diags.HasError() {
			return diags
		}
	}
	if monitor.Headers.IsUnknown() || monitor.Headers.IsNull() {
		monitor.Headers = types.ListNull(resource_monitor.HeadersValue{}.Type(ctx))
	} else if !monitor.Headers.IsNull() {
		var headers []resource_monitor.HeadersValue
		diags := monitor.Headers.ElementsAs(ctx, &headers, true)
		if diags.HasError() {
			return diags
		}

		for i := range headers {
			if headers[i].Key.IsUnknown() {
				headers[i].Key = types.StringNull()
			}

			if headers[i].Value.IsUnknown() {
				headers[i].Value = types.StringNull()
			}
		}

		monitor.Headers, diags = types.ListValueFrom(ctx, resource_monitor.HeadersValue{}.Type(ctx), headers)
		if diags.HasError() {
			return diags
		}
	}

	if monitor.Assertions.IsNull() || monitor.Assertions.IsUnknown() {
		monitor.Assertions = types.ListNull(resource_monitor.AssertionsValue{}.Type(ctx))
	} else {
		var assertions []resource_monitor.AssertionsValue
		diags := monitor.Assertions.ElementsAs(ctx, &assertions, true)
		if diags.HasError() {
			return diags
		}

		for i := range assertions {
			if assertions[i].Key.IsUnknown() {
				assertions[i].Key = types.StringNull()
			}
		}

		monitor.Assertions, diags = types.ListValueFrom(ctx, resource_monitor.AssertionsValue{}.Type(ctx), assertions)
		if diags.HasError() {
			return diags
		}
	}

	return nil
}
