package monitor

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestHTTPMonitorAPIObject_BoolFalseNotOmitted(t *testing.T) {
	obj := httpMonitorAPIObject{
		Name:            "test",
		URL:             "https://example.com",
		Periodicity:     "PERIODICITY_1M",
		FollowRedirects: false,
		Active:          false,
		Public:          false,
	}

	data, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	for _, field := range []string{"followRedirects", "active", "public"} {
		val, ok := raw[field]
		if !ok {
			t.Errorf("field %q omitted from JSON when false — omitempty must be removed", field)
			continue
		}
		if val != false {
			t.Errorf("field %q = %v, want false", field, val)
		}
	}
}

func TestHTTPMonitorAPIObject_ClearableScalarsNotOmitted(t *testing.T) {
	obj := httpMonitorAPIObject{
		Name:        "m",
		URL:         "https://example.com",
		Periodicity: "PERIODICITY_1M",
		Body:        "",
		Description: "",
		Timeout:     0,
		Retry:       0,
	}
	b, err := json.Marshal(obj)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	for _, field := range []string{"body", "description", "timeout", "retry"} {
		if _, ok := raw[field]; !ok {
			t.Errorf("field %q omitted from JSON when zero — omitempty must be removed", field)
		}
	}
}

func TestHTTPAPIToModel_DropsEmptyPlaceholderHeader(t *testing.T) {
	api := httpMonitorAPIObject{
		Headers: []apiHeader{{Key: "", Value: ""}},
	}
	var data httpMonitorModel
	data.Headers = types.ListNull(types.ObjectType{AttrTypes: headerObjTypes})

	diags := httpAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if data.Headers.IsNull() {
		t.Fatal("Headers is null, want empty list")
	}
	if got := len(data.Headers.Elements()); got != 0 {
		t.Errorf("Headers length = %d, want 0", got)
	}
}

func TestHTTPAPIToModel_KeepsRealHeaders(t *testing.T) {
	api := httpMonitorAPIObject{
		Headers: []apiHeader{{Key: "X-Api-Key", Value: "secret"}},
	}
	var data httpMonitorModel

	diags := httpAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if data.Headers.IsNull() {
		t.Fatal("Headers is null, want one-element list")
	}
	elems := data.Headers.Elements()
	if len(elems) != 1 {
		t.Fatalf("Headers length = %d, want 1", len(elems))
	}
	obj, ok := elems[0].(types.Object)
	if !ok {
		t.Fatalf("element is %T, want types.Object", elems[0])
	}
	attrs := obj.Attributes()
	key, ok := attrs["key"].(types.String)
	if !ok {
		t.Fatalf("key is %T, want types.String", attrs["key"])
	}
	if got := key.ValueString(); got != "X-Api-Key" {
		t.Errorf("key = %q, want %q", got, "X-Api-Key")
	}
	value, ok := attrs["value"].(types.String)
	if !ok {
		t.Fatalf("value is %T, want types.String", attrs["value"])
	}
	if got := value.ValueString(); got != "secret" {
		t.Errorf("value = %q, want %q", got, "secret")
	}
}

func TestHTTPAPIToModel_KeepsRealHeadersAlongsidePlaceholder(t *testing.T) {
	api := httpMonitorAPIObject{
		Headers: []apiHeader{
			{Key: "", Value: ""},
			{Key: "X", Value: "Y"},
			{Key: "", Value: ""},
		},
	}
	var data httpMonitorModel

	diags := httpAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	elems := data.Headers.Elements()
	if len(elems) != 1 {
		t.Fatalf("Headers length = %d, want 1 (placeholders dropped)", len(elems))
	}
	obj, ok := elems[0].(types.Object)
	if !ok {
		t.Fatalf("element is %T, want types.Object", elems[0])
	}
	attrs := obj.Attributes()
	key, ok := attrs["key"].(types.String)
	if !ok {
		t.Fatalf("key is %T, want types.String", attrs["key"])
	}
	if got := key.ValueString(); got != "X" {
		t.Errorf("key = %q, want %q", got, "X")
	}
	value, ok := attrs["value"].(types.String)
	if !ok {
		t.Fatalf("value is %T, want types.String", attrs["value"])
	}
	if got := value.ValueString(); got != "Y" {
		t.Errorf("value = %q, want %q", got, "Y")
	}
}

func TestHTTPAPIToModel_HeaderListShapeMatchesGetForAbsentBlock(t *testing.T) {
	// Sanity: when the API returns nothing and state had no headers, the
	// mapper should produce an empty (non-null) list — matching the shape
	// Get() produces for an absent ListNestedBlock. Guards #19 from
	// regressing if someone "optimizes" the filter back to a null branch.
	api := httpMonitorAPIObject{Headers: nil}
	var data httpMonitorModel

	diags := httpAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if data.Headers.IsNull() {
		t.Fatal("Headers is null, want empty list to match plan shape")
	}
	if got := len(data.Headers.Elements()); got != 0 {
		t.Errorf("Headers length = %d, want 0", got)
	}
}

func TestGetMonitorResponse_NestedMonitorWrapper(t *testing.T) {
	// Simulates the actual API v2 response format
	apiJSON := `{"monitor":{"http":{"name":"Hub UI","url":"https://hub.traefik.io","followRedirects":false}}}`

	var resp getMonitorResponse
	if err := json.Unmarshal([]byte(apiJSON), &resp); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if resp.Monitor.HTTP == nil {
		t.Fatal("expected monitor.http to be parsed, got nil")
	}
	if resp.Monitor.HTTP.Name != "Hub UI" {
		t.Errorf("name = %q, want %q", resp.Monitor.HTTP.Name, "Hub UI")
	}
	if resp.Monitor.HTTP.FollowRedirects != false {
		t.Error("followRedirects = true, want false")
	}
}
