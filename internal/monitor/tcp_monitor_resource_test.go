package monitor

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestTCPModelToAPI_DegradedAtPresence(t *testing.T) {
	data := tcpMonitorModel{
		Name:        types.StringValue("m"),
		URI:         types.StringValue("db.example.com:5432"),
		Periodicity: types.StringValue("1m"),
		DegradedAt:  types.Int64Value(0),
	}

	got, diags := tcpModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.HasDegradedAt() {
		t.Error("degraded_at must stay absent when zero")
	}

	data.DegradedAt = types.Int64Value(15000)
	got, diags = tcpModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !got.HasDegradedAt() || got.GetDegradedAt() != 15000 {
		t.Errorf("degraded_at = %d (present=%v), want 15000 present", got.GetDegradedAt(), got.HasDegradedAt())
	}
}

func TestTCPModelToAPI_SendsZeroScalars(t *testing.T) {
	got, diags := tcpModelToAPI(context.Background(), tcpMonitorModel{
		Name:        types.StringValue("m"),
		URI:         types.StringValue("db.example.com:5432"),
		Periodicity: types.StringValue("1m"),
		Timeout:     types.Int64Value(0),
		Retry:       types.Int64Value(0),
		Active:      types.BoolValue(false),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.GetTimeout() != 0 || got.GetRetry() != 0 || got.GetActive() {
		t.Error("zero-valued scalars should round-trip as zero")
	}
}

func TestTCPAPIToModel_RoundTrip(t *testing.T) {
	api, diags := tcpModelToAPI(context.Background(), tcpMonitorModel{
		Name:        types.StringValue("Database"),
		URI:         types.StringValue("db.example.com:5432"),
		Periodicity: types.StringValue("5m"),
		Timeout:     types.Int64Value(10000),
		Active:      types.BoolValue(true),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	var data tcpMonitorModel
	if diags := tcpAPIToModel(context.Background(), api, &data); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if data.URI.ValueString() != "db.example.com:5432" {
		t.Errorf("URI = %q, want db.example.com:5432", data.URI.ValueString())
	}
	if data.Periodicity.ValueString() != "5m" {
		t.Errorf("Periodicity = %q, want 5m", data.Periodicity.ValueString())
	}
	if data.Timeout.ValueInt64() != 10000 {
		t.Errorf("Timeout = %d, want 10000", data.Timeout.ValueInt64())
	}
}
