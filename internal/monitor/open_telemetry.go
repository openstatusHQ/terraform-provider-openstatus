package monitor

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// apiHeader is shared with the HTTP monitor's top-level headers block —
// both come from the proto `Headers` message.
type apiHeader struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type apiOpenTelemetry struct {
	Endpoint string      `json:"endpoint"`
	Headers  []apiHeader `json:"headers,omitempty"`
}

var openTelemetryHeaderObjTypes = map[string]attr.Type{
	"key":   types.StringType,
	"value": types.StringType,
}

var openTelemetryObjTypes = map[string]attr.Type{
	"endpoint": types.StringType,
	"headers":  types.ListType{ElemType: types.ObjectType{AttrTypes: openTelemetryHeaderObjTypes}},
}

func openTelemetrySchemaBlock() schema.Block {
	return schema.SingleNestedBlock{
		MarkdownDescription: "OpenTelemetry exporter configuration.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				Optional:   true,
				Validators: []validator.String{stringvalidator.LengthAtMost(2048)},
			},
		},
		Blocks: map[string]schema.Block{
			"headers": schema.ListNestedBlock{
				Validators: []validator.List{listvalidator.SizeAtMost(20)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"key":   schema.StringAttribute{Required: true},
						"value": schema.StringAttribute{Required: true},
					},
				},
			},
		},
	}
}

type otelHeaderTF struct {
	Key   string `tfsdk:"key"`
	Value string `tfsdk:"value"`
}

type otelTF struct {
	Endpoint string         `tfsdk:"endpoint"`
	Headers  []otelHeaderTF `tfsdk:"headers"`
}

func openTelemetryToAPI(ctx context.Context, obj types.Object) (*apiOpenTelemetry, diag.Diagnostics) {
	var diags diag.Diagnostics
	if obj.IsNull() || obj.IsUnknown() {
		return nil, diags
	}
	var src otelTF
	diags.Append(obj.As(ctx, &src, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil, diags
	}
	out := &apiOpenTelemetry{Endpoint: src.Endpoint}
	for _, h := range src.Headers {
		out.Headers = append(out.Headers, apiHeader(h))
	}
	return out, diags
}

func openTelemetryFromAPI(_ context.Context, api *apiOpenTelemetry) (types.Object, diag.Diagnostics) {
	var diags diag.Diagnostics
	if api == nil {
		return types.ObjectNull(openTelemetryObjTypes), diags
	}
	headerObjs := make([]attr.Value, 0, len(api.Headers))
	for _, h := range api.Headers {
		hObj, d := types.ObjectValue(openTelemetryHeaderObjTypes, map[string]attr.Value{
			"key":   types.StringValue(h.Key),
			"value": types.StringValue(h.Value),
		})
		diags.Append(d...)
		headerObjs = append(headerObjs, hObj)
	}
	headersList, d := types.ListValue(types.ObjectType{AttrTypes: openTelemetryHeaderObjTypes}, headerObjs)
	diags.Append(d...)
	obj, d := types.ObjectValue(openTelemetryObjTypes, map[string]attr.Value{
		"endpoint": types.StringValue(api.Endpoint),
		"headers":  headersList,
	})
	diags.Append(d...)
	return obj, diags
}
