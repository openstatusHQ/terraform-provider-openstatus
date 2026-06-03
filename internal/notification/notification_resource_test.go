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

func TestExtractProviderData_NtfyOmitsOptionalFields(t *testing.T) {
	// Optional ntfy attributes (server_url, token) must round-trip when
	// omitted in HCL. Before the fix, ElementsAs into a plain `string`
	// failed with "Received null value, however the target type cannot
	// handle null values" → terraform apply aborted with a Value
	// Conversion Error.
	attrTypes := map[string]attr.Type{
		"topic":      types.StringType,
		"server_url": types.StringType,
		"token":      types.StringType,
	}
	obj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
		"topic":      types.StringValue("alerts"),
		"server_url": types.StringNull(),
		"token":      types.StringNull(),
	})
	list, _ := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{obj})

	model := notificationModel{
		ProviderType: types.StringValue("ntfy"),
		Ntfy:         list,
	}
	got, diags := extractProviderData(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	inner, ok := got["ntfy"].(map[string]interface{})
	if !ok {
		t.Fatalf("ntfy payload type = %T, want map", got["ntfy"])
	}
	if inner["topic"] != "alerts" {
		t.Errorf("topic = %v, want alerts", inner["topic"])
	}
	if _, ok := inner["serverUrl"]; ok {
		t.Errorf("serverUrl was sent despite null source; got=%v", inner["serverUrl"])
	}
	if _, ok := inner["token"]; ok {
		t.Errorf("token was sent despite null source; got=%v", inner["token"])
	}
}

func TestExtractProviderData_NtfyKeepsOptionalFields(t *testing.T) {
	attrTypes := map[string]attr.Type{
		"topic":      types.StringType,
		"server_url": types.StringType,
		"token":      types.StringType,
	}
	obj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
		"topic":      types.StringValue("alerts"),
		"server_url": types.StringValue("https://ntfy.example.com"),
		"token":      types.StringValue("tk_123"),
	})
	list, _ := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{obj})

	model := notificationModel{
		ProviderType: types.StringValue("ntfy"),
		Ntfy:         list,
	}
	got, diags := extractProviderData(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	inner, ok := got["ntfy"].(map[string]interface{})
	if !ok {
		t.Fatalf("ntfy payload type = %T, want map", got["ntfy"])
	}
	if inner["serverUrl"] != "https://ntfy.example.com" {
		t.Errorf("serverUrl = %v, want https://ntfy.example.com", inner["serverUrl"])
	}
	if inner["token"] != "tk_123" {
		t.Errorf("token = %v, want tk_123", inner["token"])
	}
}

func TestNotificationAPIToModel_NtfyKeepsNullTokenWhenOmittedFromHCL(t *testing.T) {
	// User's HCL has no `token = ...` line. Plan token is null. API doesn't
	// echo token. Post-apply state token MUST be null (not "") — otherwise
	// Terraform aborts with "inconsistent values for sensitive attribute".
	attrTypes := map[string]attr.Type{
		"topic":      types.StringType,
		"server_url": types.StringType,
		"token":      types.StringType,
	}
	plannedObj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
		"topic":      types.StringValue("openstatus"),
		"server_url": types.StringValue("https://ntfy.sh"),
		"token":      types.StringNull(),
	})
	plannedList, _ := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{plannedObj})

	api := apiNotification{
		ID:       "1128",
		Provider: "NOTIFICATION_PROVIDER_NTFY",
		Data: map[string]interface{}{
			"ntfy": map[string]interface{}{
				"topic":     "openstatus",
				"serverUrl": "https://ntfy.sh",
			},
		},
	}
	data := notificationModel{Ntfy: plannedList}

	diags := notificationAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	obj, ok := data.Ntfy.Elements()[0].(types.Object)
	if !ok {
		t.Fatalf("element is %T, want types.Object", data.Ntfy.Elements()[0])
	}
	tok, ok := obj.Attributes()["token"].(types.String)
	if !ok {
		t.Fatalf("token is %T, want types.String", obj.Attributes()["token"])
	}
	if !tok.IsNull() {
		t.Errorf("token = %q (IsNull=false), want null to match planned null shape", tok.ValueString())
	}
}

func TestNotificationAPIToModel_NtfyPreservesTokenWhenAPIStripsIt(t *testing.T) {
	// The OpenStatus API does not echo back ntfy.token (Sensitive). On
	// Update, the mapper must preserve the value already in data (loaded
	// from Plan) so the post-apply state matches what Terraform planned —
	// otherwise apply fails with "inconsistent values for sensitive
	// attribute".
	attrTypes := map[string]attr.Type{
		"topic":      types.StringType,
		"server_url": types.StringType,
		"token":      types.StringType,
	}
	plannedObj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
		"topic":      types.StringValue("alerts"),
		"server_url": types.StringValue("https://ntfy.example.com"),
		"token":      types.StringValue("tk_planned"),
	})
	plannedList, _ := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{plannedObj})

	api := apiNotification{
		ID:       "1",
		Name:     "ntfy",
		Provider: "NOTIFICATION_PROVIDER_NTFY",
		Data: map[string]interface{}{
			"ntfy": map[string]interface{}{
				"topic":     "alerts",
				"serverUrl": "https://ntfy.example.com",
				// token intentionally omitted, matching real API behavior.
			},
		},
	}
	data := notificationModel{Ntfy: plannedList}

	diags := notificationAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	elems := data.Ntfy.Elements()
	if len(elems) != 1 {
		t.Fatalf("Ntfy length = %d, want 1", len(elems))
	}
	obj, ok := elems[0].(types.Object)
	if !ok {
		t.Fatalf("element is %T, want types.Object", elems[0])
	}
	tok, ok := obj.Attributes()["token"].(types.String)
	if !ok {
		t.Fatalf("token is %T, want types.String", obj.Attributes()["token"])
	}
	if got := tok.ValueString(); got != "tk_planned" {
		t.Errorf("token = %q, want %q (the planned value was lost)", got, "tk_planned")
	}
}

func TestNotificationAPIToModel_NtfyUsesAPITokenIfPresent(t *testing.T) {
	// If the API ever starts echoing the token back, that takes priority
	// over whatever data already holds.
	api := apiNotification{
		ID:       "1",
		Provider: "NOTIFICATION_PROVIDER_NTFY",
		Data: map[string]interface{}{
			"ntfy": map[string]interface{}{
				"topic": "alerts",
				"token": "tk_api",
			},
		},
	}
	var data notificationModel
	diags := notificationAPIToModel(context.Background(), api, &data)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	obj, ok := data.Ntfy.Elements()[0].(types.Object)
	if !ok {
		t.Fatalf("element is %T, want types.Object", data.Ntfy.Elements()[0])
	}
	tok, ok := obj.Attributes()["token"].(types.String)
	if !ok {
		t.Fatalf("token is %T, want types.String", obj.Attributes()["token"])
	}
	if got := tok.ValueString(); got != "tk_api" {
		t.Errorf("token = %q, want %q", got, "tk_api")
	}
}

func rawKeys(m map[string]json.RawMessage) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
