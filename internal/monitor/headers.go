package monitor

import (
	"context"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var headerObjTypes = map[string]attr.Type{
	"key":   types.StringType,
	"value": types.StringType,
}

// headersSchemaBlock is the key/value list shared by HTTP `headers` and gRPC
// `metadata`. Values often carry credentials, so they are marked sensitive.
func headersSchemaBlock(description string) schema.Block {
	return schema.ListNestedBlock{
		MarkdownDescription: description,
		Validators:          []validator.List{listvalidator.SizeAtMost(20)},
		NestedObject: schema.NestedBlockObject{
			Attributes: map[string]schema.Attribute{
				"key":   schema.StringAttribute{Required: true},
				"value": schema.StringAttribute{Required: true, Sensitive: true},
			},
		},
	}
}

func headersToAPI(ctx context.Context, list types.List) ([]*monitorv1.Headers, diag.Diagnostics) {
	var diags diag.Diagnostics
	if list.IsNull() || list.IsUnknown() {
		return nil, diags
	}

	var tfHeaders []struct {
		Key   string `tfsdk:"key"`
		Value string `tfsdk:"value"`
	}
	diags.Append(list.ElementsAs(ctx, &tfHeaders, false)...)

	headers := make([]*monitorv1.Headers, 0, len(tfHeaders))
	for _, h := range tfHeaders {
		header := &monitorv1.Headers{}
		header.SetKey(h.Key)
		header.SetValue(h.Value)
		headers = append(headers, header)
	}
	return headers, diags
}

func headersFromAPI(api []*monitorv1.Headers) (types.List, diag.Diagnostics) {
	var diags diag.Diagnostics

	// API returns `headers: [{}]` as a placeholder when no headers are
	// configured. Drop empty entries so block count matches the plan (#19).
	objs := make([]attr.Value, 0, len(api))
	for _, h := range api {
		if h.GetKey() == "" && h.GetValue() == "" {
			continue
		}
		obj, d := types.ObjectValue(headerObjTypes, map[string]attr.Value{
			"key":   types.StringValue(h.GetKey()),
			"value": types.StringValue(h.GetValue()),
		})
		diags.Append(d...)
		objs = append(objs, obj)
	}
	list, d := types.ListValue(types.ObjectType{AttrTypes: headerObjTypes}, objs)
	diags.Append(d...)
	return list, diags
}
