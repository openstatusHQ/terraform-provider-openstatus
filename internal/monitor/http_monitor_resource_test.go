package monitor

import (
	"context"
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func httpMonitorWithHeaders(headers ...*monitorv1.Headers) *monitorv1.HTTPMonitor {
	m := &monitorv1.HTTPMonitor{}
	m.SetHeaders(headers)
	return m
}

func header(key, value string) *monitorv1.Headers {
	h := &monitorv1.Headers{}
	h.SetKey(key)
	h.SetValue(value)
	return h
}

// follow_redirects and degraded_at are proto-optional, so presence is the
// provider's responsibility rather than a JSON encoding detail.
func TestHTTPModelToAPI_OptionalFieldPresence(t *testing.T) {
	data := httpMonitorModel{
		Name:            types.StringValue("m"),
		URL:             types.StringValue("https://example.com"),
		Periodicity:     types.StringValue("1m"),
		Method:          types.StringValue("GET"),
		FollowRedirects: types.BoolValue(false),
		DegradedAt:      types.Int64Value(0),
	}

	got, diags := httpModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	if !got.HasFollowRedirects() {
		t.Error("follow_redirects must be present even when false, otherwise the server keeps its own default")
	}
	if got.GetFollowRedirects() {
		t.Error("follow_redirects = true, want false")
	}
	if got.HasDegradedAt() {
		t.Error("degraded_at must stay absent when zero, matching the previous omitempty behaviour")
	}

	data.DegradedAt = types.Int64Value(30000)
	got, diags = httpModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !got.HasDegradedAt() || got.GetDegradedAt() != 30000 {
		t.Errorf("degraded_at = %d (present=%v), want 30000 present", got.GetDegradedAt(), got.HasDegradedAt())
	}
}

func TestHTTPModelToAPI_RejectsUnknownEnums(t *testing.T) {
	_, diags := httpModelToAPI(context.Background(), httpMonitorModel{
		Periodicity: types.StringValue("7m"),
	})
	if !diags.HasError() {
		t.Error("expected an error for an unknown periodicity")
	}

	_, diags = httpModelToAPI(context.Background(), httpMonitorModel{
		Periodicity: types.StringValue("1m"),
		Method:      types.StringValue("BREW"),
	})
	if !diags.HasError() {
		t.Error("expected an error for an unknown method")
	}
}

func TestHTTPAPIToModel_DropsEmptyPlaceholderHeader(t *testing.T) {
	var data httpMonitorModel
	data.Headers = types.ListNull(types.ObjectType{AttrTypes: headerObjTypes})

	diags := httpAPIToModel(context.Background(), httpMonitorWithHeaders(header("", "")), &data)
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
	var data httpMonitorModel

	diags := httpAPIToModel(context.Background(), httpMonitorWithHeaders(header("X-Api-Key", "secret")), &data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	elems := data.Headers.Elements()
	if len(elems) != 1 {
		t.Fatalf("Headers length = %d, want 1", len(elems))
	}
	attrs := elems[0].(types.Object).Attributes()
	if got := attrs["key"].(types.String).ValueString(); got != "X-Api-Key" {
		t.Errorf("key = %q, want X-Api-Key", got)
	}
	if got := attrs["value"].(types.String).ValueString(); got != "secret" {
		t.Errorf("value = %q, want secret", got)
	}
}

func TestHTTPAPIToModel_KeepsRealHeadersAlongsidePlaceholder(t *testing.T) {
	var data httpMonitorModel

	api := httpMonitorWithHeaders(header("", ""), header("X", "Y"), header("", ""))
	diags := httpAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	elems := data.Headers.Elements()
	if len(elems) != 1 {
		t.Fatalf("Headers length = %d, want 1 (placeholders dropped)", len(elems))
	}
	attrs := elems[0].(types.Object).Attributes()
	if got := attrs["key"].(types.String).ValueString(); got != "X" {
		t.Errorf("key = %q, want X", got)
	}
	if got := attrs["value"].(types.String).ValueString(); got != "Y" {
		t.Errorf("value = %q, want Y", got)
	}
}

func TestHTTPAPIToModel_HeaderListShapeMatchesGetForAbsentBlock(t *testing.T) {
	// Sanity: when the API returns nothing and state had no headers, the
	// mapper should produce an empty (non-null) list — matching the shape
	// Get() produces for an absent ListNestedBlock. Guards #19 from
	// regressing if someone "optimizes" the filter back to a null branch.
	var data httpMonitorModel

	diags := httpAPIToModel(context.Background(), &monitorv1.HTTPMonitor{}, &data)
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

func TestHTTPAPIToModel_RoundTripsAssertions(t *testing.T) {
	statusAssertion := &monitorv1.StatusCodeAssertion{}
	statusAssertion.SetTarget(200)
	statusAssertion.SetComparator(monitorv1.NumberComparator_NUMBER_COMPARATOR_EQUAL)

	bodyAssertion := &monitorv1.BodyAssertion{}
	bodyAssertion.SetTarget("ok")
	bodyAssertion.SetComparator(monitorv1.StringComparator_STRING_COMPARATOR_CONTAINS)

	headerAssertion := &monitorv1.HeaderAssertion{}
	headerAssertion.SetKey("X-Cache")
	headerAssertion.SetTarget("HIT")
	headerAssertion.SetComparator(monitorv1.StringComparator_STRING_COMPARATOR_EQUAL)

	api := &monitorv1.HTTPMonitor{}
	api.SetStatusCodeAssertions([]*monitorv1.StatusCodeAssertion{statusAssertion})
	api.SetBodyAssertions([]*monitorv1.BodyAssertion{bodyAssertion})
	api.SetHeaderAssertions([]*monitorv1.HeaderAssertion{headerAssertion})

	var data httpMonitorModel
	diags := httpAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	status := data.StatusCodeAssertions.Elements()[0].(types.Object).Attributes()
	if got := status["target"].(types.Int64).ValueInt64(); got != 200 {
		t.Errorf("status target = %d, want 200", got)
	}
	if got := status["comparator"].(types.String).ValueString(); got != "eq" {
		t.Errorf("status comparator = %q, want eq", got)
	}

	body := data.BodyAssertions.Elements()[0].(types.Object).Attributes()
	if got := body["comparator"].(types.String).ValueString(); got != "contains" {
		t.Errorf("body comparator = %q, want contains", got)
	}

	head := data.HeaderAssertions.Elements()[0].(types.Object).Attributes()
	if got := head["key"].(types.String).ValueString(); got != "X-Cache" {
		t.Errorf("header key = %q, want X-Cache", got)
	}
}

func TestMonitorConfigOneof(t *testing.T) {
	httpMonitor := &monitorv1.HTTPMonitor{}
	httpMonitor.SetName("Hub UI")
	httpMonitor.SetUrl("https://hub.traefik.io")
	httpMonitor.SetFollowRedirects(false)

	config := &monitorv1.MonitorConfig{}
	config.SetHttp(httpMonitor)

	if config.GetHttp() == nil {
		t.Fatal("expected the http arm to be set")
	}
	if config.GetTcp() != nil || config.GetDns() != nil {
		t.Error("only one arm of the oneof may be set")
	}
	if config.GetHttp().GetName() != "Hub UI" {
		t.Errorf("name = %q, want Hub UI", config.GetHttp().GetName())
	}
	if config.GetHttp().GetFollowRedirects() {
		t.Error("follow_redirects = true, want false")
	}
}
