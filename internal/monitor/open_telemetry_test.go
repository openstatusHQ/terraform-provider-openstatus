package monitor

import (
	"context"
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func otelObject(endpoint string, headers ...[2]string) types.Object {
	headerObjs := make([]attr.Value, 0, len(headers))
	for _, h := range headers {
		obj, _ := types.ObjectValue(openTelemetryHeaderObjTypes, map[string]attr.Value{
			"key":   types.StringValue(h[0]),
			"value": types.StringValue(h[1]),
		})
		headerObjs = append(headerObjs, obj)
	}
	list, _ := types.ListValue(types.ObjectType{AttrTypes: openTelemetryHeaderObjTypes}, headerObjs)
	obj, _ := types.ObjectValue(openTelemetryObjTypes, map[string]attr.Value{
		"endpoint": types.StringValue(endpoint),
		"headers":  list,
	})
	return obj
}

func TestOpenTelemetryToAPI_NullAndUnknownAreNil(t *testing.T) {
	got, diags := openTelemetryToAPI(context.Background(), types.ObjectNull(openTelemetryObjTypes))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got != nil {
		t.Errorf("null object should map to nil, got %v", got)
	}

	got, diags = openTelemetryToAPI(context.Background(), types.ObjectUnknown(openTelemetryObjTypes))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got != nil {
		t.Errorf("unknown object should map to nil, got %v", got)
	}
}

func TestOpenTelemetryToAPI_SetsEndpointAndHeaders(t *testing.T) {
	got, diags := openTelemetryToAPI(context.Background(),
		otelObject("https://otel.example.com/v1/metrics", [2]string{"X-Api-Key", "secret"}))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.GetEndpoint() != "https://otel.example.com/v1/metrics" {
		t.Errorf("endpoint = %q, want https://otel.example.com/v1/metrics", got.GetEndpoint())
	}
	if len(got.GetHeaders()) != 1 {
		t.Fatalf("headers length = %d, want 1", len(got.GetHeaders()))
	}
	if got.GetHeaders()[0].GetKey() != "X-Api-Key" || got.GetHeaders()[0].GetValue() != "secret" {
		t.Errorf("header = %v, want X-Api-Key/secret", got.GetHeaders()[0])
	}
}

func TestOpenTelemetryFromAPI_NilIsNullObject(t *testing.T) {
	got, diags := openTelemetryFromAPI(context.Background(), nil, types.ObjectNull(openTelemetryObjTypes))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !got.IsNull() {
		t.Errorf("nil config should map to a null object, got %v", got)
	}
}

func TestOpenTelemetryRoundTrip(t *testing.T) {
	in := otelObject("https://otel.example.com", [2]string{"a", "b"}, [2]string{"c", "d"})

	api, diags := openTelemetryToAPI(context.Background(), in)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	out, diags := openTelemetryFromAPI(context.Background(), api, in)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	if out.Attributes()["endpoint"].(types.String).ValueString() != "https://otel.example.com" {
		t.Errorf("endpoint did not round-trip, got %v", out.Attributes()["endpoint"])
	}
	headers := out.Attributes()["headers"].(types.List).Elements()
	if len(headers) != 2 {
		t.Fatalf("headers length = %d, want 2", len(headers))
	}
}

// otelObjectNullEndpoint builds an open_telemetry object whose endpoint is
// null, as produced by a block that only sets headers.
func otelObjectNullEndpoint(headers ...[2]string) types.Object {
	headerObjs := make([]attr.Value, 0, len(headers))
	for _, h := range headers {
		obj, _ := types.ObjectValue(openTelemetryHeaderObjTypes, map[string]attr.Value{
			"key":   types.StringValue(h[0]),
			"value": types.StringValue(h[1]),
		})
		headerObjs = append(headerObjs, obj)
	}
	list, _ := types.ListValue(types.ObjectType{AttrTypes: openTelemetryHeaderObjTypes}, headerObjs)
	obj, _ := types.ObjectValue(openTelemetryObjTypes, map[string]attr.Value{
		"endpoint": types.StringNull(),
		"headers":  list,
	})
	return obj
}

func TestOpenTelemetryToAPI_NullEndpoint(t *testing.T) {
	got, diags := openTelemetryToAPI(context.Background(), otelObjectNullEndpoint([2]string{"X-Api-Key", "secret"}))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.GetEndpoint() != "" {
		t.Errorf("endpoint = %q, want empty", got.GetEndpoint())
	}
	if len(got.GetHeaders()) != 1 {
		t.Fatalf("headers length = %d, want 1", len(got.GetHeaders()))
	}
}

func TestOpenTelemetryFromAPI_PreservesNullEndpoint(t *testing.T) {
	prior := otelObjectNullEndpoint([2]string{"a", "b"})

	api, diags := openTelemetryToAPI(context.Background(), prior)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	out, diags := openTelemetryFromAPI(context.Background(), api, prior)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	if !out.Attributes()["endpoint"].(types.String).IsNull() {
		t.Errorf("endpoint should stay null when unset in config, got %v", out.Attributes()["endpoint"])
	}
	if len(out.Attributes()["headers"].(types.List).Elements()) != 1 {
		t.Errorf("headers did not round-trip, got %v", out.Attributes()["headers"])
	}
}

func TestOpenTelemetryFromAPI_KeepsEndpointValueOverNullPrior(t *testing.T) {
	api := &monitorv1.OpenTelemetryConfig{}
	api.SetEndpoint("https://otel.example.com")

	out, diags := openTelemetryFromAPI(context.Background(), api, types.ObjectNull(openTelemetryObjTypes))
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if out.Attributes()["endpoint"].(types.String).ValueString() != "https://otel.example.com" {
		t.Errorf("endpoint = %v, want https://otel.example.com", out.Attributes()["endpoint"])
	}
}

func TestHTTPMonitorOpenTelemetryPresence(t *testing.T) {
	data := httpMonitorModel{
		Name:        types.StringValue("m"),
		URL:         types.StringValue("https://example.com"),
		Periodicity: types.StringValue("1m"),
		Method:      types.StringValue("GET"),
	}

	got, diags := httpModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.GetOpenTelemetry() != nil {
		t.Error("open_telemetry should stay absent when the block is omitted")
	}

	data.OpenTelemetry = otelObject("https://otel.example.com")
	got, diags = httpModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.GetOpenTelemetry() == nil {
		t.Fatal("open_telemetry should be set when the block is present")
	}
	if got.GetOpenTelemetry().GetEndpoint() != "https://otel.example.com" {
		t.Errorf("endpoint = %q, want https://otel.example.com", got.GetOpenTelemetry().GetEndpoint())
	}
}
