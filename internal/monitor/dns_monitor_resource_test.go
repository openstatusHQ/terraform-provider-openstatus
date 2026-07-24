package monitor

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func recordAssertionList(record, comparator, target string) types.List {
	obj, _ := types.ObjectValue(recordAssertionObjTypes, map[string]attr.Value{
		"record":     types.StringValue(record),
		"comparator": types.StringValue(comparator),
		"target":     types.StringValue(target),
	})
	list, _ := types.ListValue(types.ObjectType{AttrTypes: recordAssertionObjTypes}, []attr.Value{obj})
	return list
}

func TestDNSModelToAPI_DegradedAtPresence(t *testing.T) {
	data := dnsMonitorModel{
		Name:        types.StringValue("m"),
		URI:         types.StringValue("example.com"),
		Periodicity: types.StringValue("10m"),
		DegradedAt:  types.Int64Value(0),
	}

	got, diags := dnsModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.HasDegradedAt() {
		t.Error("degraded_at must stay absent when zero")
	}

	data.DegradedAt = types.Int64Value(5000)
	got, diags = dnsModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !got.HasDegradedAt() || got.GetDegradedAt() != 5000 {
		t.Errorf("degraded_at = %d (present=%v), want 5000 present", got.GetDegradedAt(), got.HasDegradedAt())
	}
}

func TestDNSModelToAPI_RejectsUnknownComparator(t *testing.T) {
	_, diags := dnsModelToAPI(context.Background(), dnsMonitorModel{
		Periodicity:      types.StringValue("10m"),
		RecordAssertions: recordAssertionList("A", "starts_with", "1.2.3.4"),
	})
	if !diags.HasError() {
		t.Error("expected an error for an unknown record comparator")
	}
}

func TestDNSAPIToModel_RecordAssertionsRoundTrip(t *testing.T) {
	api, diags := dnsModelToAPI(context.Background(), dnsMonitorModel{
		Name:             types.StringValue("DNS"),
		URI:              types.StringValue("example.com"),
		Periodicity:      types.StringValue("10m"),
		RecordAssertions: recordAssertionList("A", "eq", "93.184.216.34"),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	var data dnsMonitorModel
	if diags := dnsAPIToModel(context.Background(), api, &data); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	elems := data.RecordAssertions.Elements()
	if len(elems) != 1 {
		t.Fatalf("RecordAssertions length = %d, want 1", len(elems))
	}
	attrs := elems[0].(types.Object).Attributes()
	if got := attrs["record"].(types.String).ValueString(); got != "A" {
		t.Errorf("record = %q, want A", got)
	}
	if got := attrs["comparator"].(types.String).ValueString(); got != "eq" {
		t.Errorf("comparator = %q, want eq", got)
	}
	if got := attrs["target"].(types.String).ValueString(); got != "93.184.216.34" {
		t.Errorf("target = %q, want 93.184.216.34", got)
	}
}
