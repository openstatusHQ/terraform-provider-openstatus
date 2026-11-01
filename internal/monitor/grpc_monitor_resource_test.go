package monitor

import (
	"context"
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func grpcTestModel() grpcMonitorModel {
	return grpcMonitorModel{
		Name:        types.StringValue("m"),
		URI:         types.StringValue("grpc.example.com:443"),
		Periodicity: types.StringValue("1m"),
		TLSMode:     types.StringValue("tls"),
		Metadata:    types.ListNull(types.ObjectType{AttrTypes: headerObjTypes}),
	}
}

func TestGRPCModelToAPI_DegradedAtPresence(t *testing.T) {
	data := grpcTestModel()
	data.DegradedAt = types.Int64Value(0)

	got, diags := grpcModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.HasDegradedAt() {
		t.Error("degraded_at must stay absent when zero")
	}

	data.DegradedAt = types.Int64Value(15000)
	got, diags = grpcModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !got.HasDegradedAt() || got.GetDegradedAt() != 15000 {
		t.Errorf("degraded_at = %d (present=%v), want 15000 present", got.GetDegradedAt(), got.HasDegradedAt())
	}
}

func TestGRPCModelToAPI_SendsZeroScalars(t *testing.T) {
	data := grpcTestModel()
	data.Timeout = types.Int64Value(0)
	data.Retry = types.Int64Value(0)
	data.Active = types.BoolValue(false)

	got, diags := grpcModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.GetTimeout() != 0 || got.GetRetry() != 0 || got.GetActive() {
		t.Error("zero-valued scalars should round-trip as zero")
	}
}

func TestGRPCModelToAPI_RejectsUnknownTLSMode(t *testing.T) {
	data := grpcTestModel()
	data.TLSMode = types.StringValue("mtls")

	if _, diags := grpcModelToAPI(context.Background(), data); !diags.HasError() {
		t.Fatal("expected error for unknown tls_mode")
	}
}

// Removing tls_mode from config falls back to the "tls" default, which must
// be sent explicitly: the API ignores an unspecified tls_mode on update.
func TestGRPCModelToAPI_AlwaysSendsTLSMode(t *testing.T) {
	got, diags := grpcModelToAPI(context.Background(), grpcTestModel())
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !got.HasTlsMode() || got.GetTlsMode() != monitorv1.GRPCTlsMode_GRPC_TLS_MODE_TLS {
		t.Errorf("tls_mode = %v (present=%v), want TLS present", got.GetTlsMode(), got.HasTlsMode())
	}
}

func TestGRPCAPIToModel_RoundTrip(t *testing.T) {
	metadata, diags := types.ListValue(types.ObjectType{AttrTypes: headerObjTypes}, []attr.Value{
		types.ObjectValueMust(headerObjTypes, map[string]attr.Value{
			"key":   types.StringValue("authorization"),
			"value": types.StringValue("Bearer token"),
		}),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	data := grpcTestModel()
	data.Name = types.StringValue("Health")
	data.Periodicity = types.StringValue("5m")
	data.Timeout = types.Int64Value(10000)
	data.Service = types.StringValue("my.package.Service")
	data.TLSMode = types.StringValue("plaintext")
	data.Metadata = metadata

	api, diags := grpcModelToAPI(context.Background(), data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	var got grpcMonitorModel
	if diags := grpcAPIToModel(context.Background(), api, &got); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if got.URI.ValueString() != "grpc.example.com:443" {
		t.Errorf("URI = %q, want grpc.example.com:443", got.URI.ValueString())
	}
	if got.Periodicity.ValueString() != "5m" {
		t.Errorf("Periodicity = %q, want 5m", got.Periodicity.ValueString())
	}
	if got.Service.ValueString() != "my.package.Service" {
		t.Errorf("Service = %q, want my.package.Service", got.Service.ValueString())
	}
	if got.TLSMode.ValueString() != "plaintext" {
		t.Errorf("TLSMode = %q, want plaintext", got.TLSMode.ValueString())
	}
	if !got.Metadata.Equal(metadata) {
		t.Errorf("Metadata = %v, want %v", got.Metadata, metadata)
	}
}

func TestGRPCAPIToModel_UnsetServiceStaysNull(t *testing.T) {
	api := &monitorv1.GRPCMonitor{}
	api.SetService("")

	data := grpcMonitorModel{Service: types.StringNull()}
	if diags := grpcAPIToModel(context.Background(), api, &data); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if !data.Service.IsNull() {
		t.Errorf("Service = %v, want null", data.Service)
	}
}

func TestGRPCAPIToModel_UnspecifiedTLSModeIsTLS(t *testing.T) {
	var data grpcMonitorModel
	if diags := grpcAPIToModel(context.Background(), &monitorv1.GRPCMonitor{}, &data); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if data.TLSMode.ValueString() != "tls" {
		t.Errorf("TLSMode = %q, want tls", data.TLSMode.ValueString())
	}
}

func TestGRPCAPIToModel_DropsEmptyPlaceholderMetadata(t *testing.T) {
	api := &monitorv1.GRPCMonitor{}
	api.SetMetadata([]*monitorv1.Headers{{}})

	var data grpcMonitorModel
	if diags := grpcAPIToModel(context.Background(), api, &data); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if n := len(data.Metadata.Elements()); n != 0 {
		t.Errorf("metadata has %d entries, want 0", n)
	}
}
