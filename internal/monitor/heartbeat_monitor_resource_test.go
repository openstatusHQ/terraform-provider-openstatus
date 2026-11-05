package monitor

import (
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The API only applies fields that are present on update, so removing one from
// the configuration only takes effect if it is still sent.
func TestHeartbeatModelToAPI_AlwaysSendsOptionalFields(t *testing.T) {
	got := heartbeatModelToAPI(heartbeatMonitorModel{
		Name:   types.StringValue("m"),
		Period: types.Int64Value(3600),
		Grace:  types.Int64Value(300),
		Active: types.BoolValue(false),
		Public: types.BoolValue(false),
	})

	if !got.HasDescription() || got.GetDescription() != "" {
		t.Errorf("description = %q (present=%v), want empty and present", got.GetDescription(), got.HasDescription())
	}
	if !got.HasActive() || got.GetActive() {
		t.Errorf("active = %v (present=%v), want false and present", got.GetActive(), got.HasActive())
	}
	if !got.HasPublic() || got.GetPublic() {
		t.Errorf("public = %v (present=%v), want false and present", got.GetPublic(), got.HasPublic())
	}
}

func TestHeartbeatAPIToModel_RoundTrip(t *testing.T) {
	api := heartbeatModelToAPI(heartbeatMonitorModel{
		Name:        types.StringValue("Nightly backup"),
		Period:      types.Int64Value(86400),
		Grace:       types.Int64Value(1800),
		Active:      types.BoolValue(true),
		Public:      types.BoolValue(true),
		Description: types.StringValue("Dumps the database"),
	})
	api.SetId("42")
	api.SetPingUrl("https://api.openstatus.dev/heartbeat/token")

	var got heartbeatMonitorModel
	heartbeatAPIToModel(api, &got)

	if got.ID.ValueString() != "42" {
		t.Errorf("ID = %q, want 42", got.ID.ValueString())
	}
	if got.Name.ValueString() != "Nightly backup" {
		t.Errorf("Name = %q, want Nightly backup", got.Name.ValueString())
	}
	if got.Period.ValueInt64() != 86400 {
		t.Errorf("Period = %d, want 86400", got.Period.ValueInt64())
	}
	if got.Grace.ValueInt64() != 1800 {
		t.Errorf("Grace = %d, want 1800", got.Grace.ValueInt64())
	}
	if !got.Active.ValueBool() || !got.Public.ValueBool() {
		t.Errorf("Active = %v, Public = %v, want both true", got.Active, got.Public)
	}
	if got.Description.ValueString() != "Dumps the database" {
		t.Errorf("Description = %q, want Dumps the database", got.Description.ValueString())
	}
	if got.PingURL.ValueString() != "https://api.openstatus.dev/heartbeat/token" {
		t.Errorf("PingURL = %q", got.PingURL.ValueString())
	}
}

func TestHeartbeatAPIToModel_UnsetDescriptionStaysNull(t *testing.T) {
	api := &monitorv1.HeartbeatMonitor{}
	api.SetDescription("")

	data := heartbeatMonitorModel{Description: types.StringNull()}
	heartbeatAPIToModel(api, &data)

	if !data.Description.IsNull() {
		t.Errorf("Description = %v, want null", data.Description)
	}
}

func TestHeartbeatAPIToModel_ConfiguredEmptyDescriptionStaysEmpty(t *testing.T) {
	api := &monitorv1.HeartbeatMonitor{}
	api.SetDescription("")

	data := heartbeatMonitorModel{Description: types.StringValue("")}
	heartbeatAPIToModel(api, &data)

	if data.Description.IsNull() || data.Description.ValueString() != "" {
		t.Errorf("Description = %v, want empty string", data.Description)
	}
}
