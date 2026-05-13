package notification

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestProviderTypeToWrapKey_MatchesProto(t *testing.T) {
	tests := map[string]string{
		"whatsapp":  "whatsapp",
		"pagerduty": "pagerduty",
	}
	for tfKey, wantWrap := range tests {
		if got := providerTypeToWrapKey[tfKey]; got != wantWrap {
			t.Errorf("providerTypeToWrapKey[%q] = %q, want %q", tfKey, got, wantWrap)
		}
	}
}

func TestExtractProviderData_UsesLowercaseWrapKeys(t *testing.T) {
	mkSingleField := func(field, value string) types.List {
		attrTypes := map[string]attr.Type{field: types.StringType}
		obj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
			field: types.StringValue(value),
		})
		list, _ := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{obj})
		return list
	}

	cases := []struct {
		name, wantWrap string
		model          notificationModel
	}{
		{
			name: "whatsapp", wantWrap: "whatsapp",
			model: notificationModel{
				ProviderType: types.StringValue("whatsapp"),
				WhatsApp:     mkSingleField("phone_number", "+4915773121555"),
			},
		},
		{
			name: "pagerduty", wantWrap: "pagerduty",
			model: notificationModel{
				ProviderType: types.StringValue("pagerduty"),
				PagerDuty:    mkSingleField("integration_key", "abc123"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := extractProviderData(context.Background(), tc.model)
			if diags.HasError() {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if _, ok := got[tc.wantWrap]; !ok {
				t.Errorf("payload missing %q wrap key; got keys=%v", tc.wantWrap, keysOf(got))
			}
		})
	}
}

func TestNotificationAPIToModel_WhatsAppRoundTrip(t *testing.T) {
	api := apiNotification{
		ID:       "1",
		Name:     "Test",
		Provider: "NOTIFICATION_PROVIDER_WHATSAPP",
		Data: map[string]interface{}{
			"whatsapp": map[string]interface{}{
				"phoneNumber": "+4915773121555",
			},
		},
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-01T00:00:00Z",
	}
	var data notificationModel
	diags := notificationAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if data.WhatsApp.IsNull() {
		t.Fatalf("WhatsApp list is null; should hold one element")
	}
	elems := data.WhatsApp.Elements()
	if len(elems) != 1 {
		t.Fatalf("expected 1 whatsapp element, got %d", len(elems))
	}
	obj := elems[0].(types.Object)
	pn := obj.Attributes()["phone_number"].(types.String).ValueString()
	if pn != "+4915773121555" {
		t.Errorf("phone_number = %q, want +4915773121555 (read-side wrap key broken)", pn)
	}
}

func keysOf(m map[string]interface{}) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}

func TestAPINotificationUpdateRequest_AlwaysSendsUpdateMonitorIDs(t *testing.T) {
	req := apiNotificationUpdateRequest{
		ID:               "abc",
		Name:             "Test",
		MonitorIDs:       []string{},
		UpdateMonitorIDs: true,
	}
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	flag, ok := raw["updateMonitorIds"]
	if !ok {
		t.Fatalf("updateMonitorIds missing from JSON; got keys=%v", rawKeys(raw))
	}
	if string(flag) != "true" {
		t.Errorf("updateMonitorIds = %s, want true", string(flag))
	}
	ids, ok := raw["monitorIds"]
	if !ok {
		t.Fatalf("monitorIds missing from JSON (omitempty dropped empty slice); got keys=%v", rawKeys(raw))
	}
	if string(ids) != "[]" {
		t.Errorf("monitorIds = %s, want []", string(ids))
	}
}

func TestExtractProviderData_MSTeams(t *testing.T) {
	attrTypes := map[string]attr.Type{"webhook_url": types.StringType}
	obj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
		"webhook_url": types.StringValue("https://example.com/teams"),
	})
	list, _ := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{obj})

	model := notificationModel{
		ProviderType: types.StringValue("ms_teams"),
		MSTeams:      list,
	}
	got, diags := extractProviderData(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	inner, ok := got["msTeams"]
	if !ok {
		t.Fatalf("payload missing msTeams wrap key; got=%v", keysOf(got))
	}
	innerMap, ok := inner.(map[string]interface{})
	if !ok {
		t.Fatalf("msTeams value type = %T, want map", inner)
	}
	if innerMap["webhookUrl"] != "https://example.com/teams" {
		t.Errorf("webhookUrl = %v, want https://example.com/teams", innerMap["webhookUrl"])
	}
}

func TestNotificationAPIToModel_MSTeamsRoundTrip(t *testing.T) {
	api := apiNotification{
		ID:       "5",
		Name:     "Teams",
		Provider: "NOTIFICATION_PROVIDER_MS_TEAMS",
		Data: map[string]interface{}{
			"msTeams": map[string]interface{}{
				"webhookUrl": "https://example.com/teams",
			},
		},
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-01T00:00:00Z",
	}
	var data notificationModel
	diags := notificationAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected error diags: %v", diags)
	}
	if data.ProviderType.ValueString() != "ms_teams" {
		t.Errorf("ProviderType = %q, want ms_teams", data.ProviderType.ValueString())
	}
	if data.MSTeams.IsNull() {
		t.Fatalf("MSTeams list is null; should hold one element")
	}
	elems := data.MSTeams.Elements()
	if len(elems) != 1 {
		t.Fatalf("expected 1 ms_teams element, got %d", len(elems))
	}
	obj := elems[0].(types.Object)
	url := obj.Attributes()["webhook_url"].(types.String).ValueString()
	if url != "https://example.com/teams" {
		t.Errorf("webhook_url = %q, want https://example.com/teams", url)
	}
}

func TestNotificationAPIToModel_UnknownProviderWarns(t *testing.T) {
	api := apiNotification{
		ID:       "7",
		Name:     "Future",
		Provider: "NOTIFICATION_PROVIDER_FUTURE",
		Data:     map[string]interface{}{},
	}
	var data notificationModel
	diags := notificationAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected error diags: %v", diags)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("warnings = %d, want 1; all diags: %v", diags.WarningsCount(), diags)
	}
	w := diags.Warnings()[0]
	if !strings.Contains(w.Detail(), "NOTIFICATION_PROVIDER_FUTURE") {
		t.Errorf("warning detail = %q; want it to mention the unknown provider", w.Detail())
	}
}

func TestNotificationAPIToModel_UnknownOpsgenieRegionWarns(t *testing.T) {
	api := apiNotification{
		ID:       "8",
		Name:     "Opsgenie",
		Provider: "NOTIFICATION_PROVIDER_OPSGENIE",
		Data: map[string]interface{}{
			"opsgenie": map[string]interface{}{
				"apiKey": "k",
				"region": "OPSGENIE_REGION_MARS",
			},
		},
	}
	var data notificationModel
	diags := notificationAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected error diags: %v", diags)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("warnings = %d, want 1; all diags: %v", diags.WarningsCount(), diags)
	}
	w := diags.Warnings()[0]
	if !strings.Contains(w.Detail(), "OPSGENIE_REGION_MARS") {
		t.Errorf("warning detail = %q; want it to mention the unknown region", w.Detail())
	}
}

func rawKeys(m map[string]json.RawMessage) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
