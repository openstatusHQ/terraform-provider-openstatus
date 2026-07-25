package privatelocation

import (
	"context"
	"testing"

	privatelocationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Terraform owns both association fields, so the sentinels must always be set:
// without them the API preserves whatever it already has and removing a monitor
// or a label from the configuration would silently never apply.
func TestNewUpdateRequest_AlwaysSendsBothSentinels(t *testing.T) {
	req := newUpdateRequest("pl_1", "EU", []string{}, map[string]string{})

	if !req.HasUpdateMonitorIds() || !req.GetUpdateMonitorIds() {
		t.Error("update_monitor_ids must be present and true")
	}
	if !req.HasUpdateMetadata() || !req.GetUpdateMetadata() {
		t.Error("update_metadata must be present and true")
	}
	if req.GetMonitorIds() == nil {
		t.Error("monitor_ids must be set (empty clears all associations)")
	}
	if req.GetMetadata() == nil {
		t.Error("metadata must be set (empty clears all labels)")
	}
	if !req.HasName() {
		t.Error("name must be present")
	}
}

func TestCollectAssociations_NullMeansEmpty(t *testing.T) {
	monitorIDs, metadata, diags := collectAssociations(context.Background(), privateLocationModel{
		MonitorIDs: types.SetNull(types.StringType),
		Metadata:   types.MapNull(types.StringType),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(monitorIDs) != 0 {
		t.Errorf("monitor_ids = %v, want empty so the API clears associations", monitorIDs)
	}
	if len(metadata) != 0 {
		t.Errorf("metadata = %v, want empty so the API clears labels", metadata)
	}
}

func TestCollectAssociations_ReadsValues(t *testing.T) {
	set, _ := types.SetValueFrom(context.Background(), types.StringType, []string{"mon_1", "mon_2"})
	m, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{"env": "prod"})

	monitorIDs, metadata, diags := collectAssociations(context.Background(), privateLocationModel{
		MonitorIDs: set,
		Metadata:   m,
	})
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if len(monitorIDs) != 2 {
		t.Errorf("monitor_ids = %v, want two entries", monitorIDs)
	}
	if metadata["env"] != "prod" {
		t.Errorf("metadata = %v, want env=prod", metadata)
	}
}

func privateLocation(id, name string) *privatelocationv1.PrivateLocation {
	l := &privatelocationv1.PrivateLocation{}
	l.SetId(id)
	l.SetName(name)
	l.SetToken("tk_secret")
	l.SetCreatedAt("2026-01-01T00:00:00Z")
	l.SetUpdatedAt("2026-01-02T00:00:00Z")
	return l
}

func TestPrivateLocationAPIToModel_SetsFields(t *testing.T) {
	api := privateLocation("pl_1", "EU Datacenter")
	api.SetMonitorIds([]string{"mon_1"})
	api.SetMetadata(map[string]string{"env": "prod"})

	var data privateLocationModel
	if diags := privateLocationAPIToModel(context.Background(), api, &data); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	if data.ID.ValueString() != "pl_1" {
		t.Errorf("ID = %q, want pl_1", data.ID.ValueString())
	}
	if data.Name.ValueString() != "EU Datacenter" {
		t.Errorf("Name = %q, want EU Datacenter", data.Name.ValueString())
	}
	if data.Token.ValueString() != "tk_secret" {
		t.Errorf("Token = %q, want tk_secret", data.Token.ValueString())
	}
	if len(data.MonitorIDs.Elements()) != 1 {
		t.Errorf("MonitorIDs = %v, want one entry", data.MonitorIDs)
	}
	if len(data.Metadata.Elements()) != 1 {
		t.Errorf("Metadata = %v, want one entry", data.Metadata)
	}
}

// Empty collections map to null rather than an empty collection, so a config
// that omits the attribute matches the state written back after apply.
func TestPrivateLocationAPIToModel_EmptyCollectionsAreNull(t *testing.T) {
	var data privateLocationModel
	if diags := privateLocationAPIToModel(context.Background(), privateLocation("pl_1", "EU"), &data); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	if !data.MonitorIDs.IsNull() {
		t.Errorf("MonitorIDs = %v, want null", data.MonitorIDs)
	}
	if !data.Metadata.IsNull() {
		t.Errorf("Metadata = %v, want null", data.Metadata)
	}
}

// The other half of the shape contract: `monitor_ids = []` and `metadata = {}`
// must survive the round trip as empty rather than collapsing to null.
// Terraform compares post-apply state to the plan byte-for-byte for these
// Optional, non-Computed attributes.
func TestPrivateLocationAPIToModel_PreservesPlannedEmptyCollections(t *testing.T) {
	ctx := context.Background()
	emptySet, _ := types.SetValueFrom(ctx, types.StringType, []string{})
	emptyMap, _ := types.MapValueFrom(ctx, types.StringType, map[string]string{})

	data := privateLocationModel{MonitorIDs: emptySet, Metadata: emptyMap}
	if diags := privateLocationAPIToModel(ctx, privateLocation("pl_1", "EU"), &data); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	if data.MonitorIDs.IsNull() || len(data.MonitorIDs.Elements()) != 0 {
		t.Errorf("MonitorIDs = %v, want a known empty set", data.MonitorIDs)
	}
	if data.Metadata.IsNull() || len(data.Metadata.Elements()) != 0 {
		t.Errorf("Metadata = %v, want a known empty map", data.Metadata)
	}
}

func TestPrivateLocationAPIToModel_UnknownCollectionsCollapseToNull(t *testing.T) {
	data := privateLocationModel{
		MonitorIDs: types.SetUnknown(types.StringType),
		Metadata:   types.MapUnknown(types.StringType),
	}
	if diags := privateLocationAPIToModel(context.Background(), privateLocation("pl_1", "EU"), &data); diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}

	if !data.MonitorIDs.IsNull() {
		t.Errorf("MonitorIDs = %v, want null", data.MonitorIDs)
	}
	if !data.Metadata.IsNull() {
		t.Errorf("Metadata = %v, want null", data.Metadata)
	}
}

func TestMapStatusFromAPI(t *testing.T) {
	cases := map[privatelocationv1.PrivateLocationStatus]string{
		privatelocationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_ACTIVE:      "active",
		privatelocationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_ERROR:       "error",
		privatelocationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_UNSPECIFIED: "unknown",
		privatelocationv1.PrivateLocationStatus(99):                                 "unknown",
	}
	for in, want := range cases {
		if got := MapStatusFromAPI(in); got != want {
			t.Errorf("MapStatusFromAPI(%v) = %q, want %q", in, got, want)
		}
	}
}

func TestStatusMapCoversEveryGeneratedValue(t *testing.T) {
	for value, name := range privatelocationv1.PrivateLocationStatus_name {
		status := privatelocationv1.PrivateLocationStatus(value)
		if status == privatelocationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_UNSPECIFIED {
			continue
		}
		if _, ok := statusFromAPI[status]; !ok {
			t.Errorf("generated status %s is not mapped to a Terraform value", name)
		}
	}
}

func TestOptionalTimestamp(t *testing.T) {
	if got := optionalTimestamp(""); !got.IsNull() {
		t.Errorf("empty timestamp = %v, want null", got)
	}
	if got := optionalTimestamp("2026-01-01T00:00:00Z"); got.ValueString() != "2026-01-01T00:00:00Z" {
		t.Errorf("timestamp = %q, want it preserved", got.ValueString())
	}
}
