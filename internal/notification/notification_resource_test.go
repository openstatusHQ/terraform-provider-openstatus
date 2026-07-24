package notification

import (
	"context"
	"strings"
	"testing"

	notificationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/notification/v1"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func singleFieldList(field, value string) types.List {
	attrTypes := map[string]attr.Type{field: types.StringType}
	obj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
		field: types.StringValue(value),
	})
	list, _ := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{obj})
	return list
}

func ntfyList(topic string, serverURL, token types.String) types.List {
	attrTypes := map[string]attr.Type{
		"topic":      types.StringType,
		"server_url": types.StringType,
		"token":      types.StringType,
	}
	obj, _ := types.ObjectValue(attrTypes, map[string]attr.Value{
		"topic":      types.StringValue(topic),
		"server_url": serverURL,
		"token":      token,
	})
	list, _ := types.ListValue(types.ObjectType{AttrTypes: attrTypes}, []attr.Value{obj})
	return list
}

func TestExtractProviderData_SelectsOneofArm(t *testing.T) {
	cases := []struct {
		name  string
		model notificationModel
		check func(*notificationv1.NotificationData) bool
	}{
		{
			name: "whatsapp",
			model: notificationModel{
				ProviderType: types.StringValue("whatsapp"),
				WhatsApp:     singleFieldList("phone_number", "+4915773121555"),
			},
			check: func(d *notificationv1.NotificationData) bool {
				return d.HasWhatsapp() && d.GetWhatsapp().GetPhoneNumber() == "+4915773121555"
			},
		},
		{
			name: "pagerduty",
			model: notificationModel{
				ProviderType: types.StringValue("pagerduty"),
				PagerDuty:    singleFieldList("integration_key", "abc123"),
			},
			check: func(d *notificationv1.NotificationData) bool {
				return d.HasPagerduty() && d.GetPagerduty().GetIntegrationKey() == "abc123"
			},
		},
		{
			name: "ms_teams",
			model: notificationModel{
				ProviderType: types.StringValue("ms_teams"),
				MSTeams:      singleFieldList("webhook_url", "https://example.com/teams"),
			},
			check: func(d *notificationv1.NotificationData) bool {
				return d.HasMsTeams() && d.GetMsTeams().GetWebhookUrl() == "https://example.com/teams"
			},
		},
		{
			name: "google_chat",
			model: notificationModel{
				ProviderType: types.StringValue("google_chat"),
				GoogleChat:   singleFieldList("webhook_url", "https://chat.googleapis.com/x"),
			},
			check: func(d *notificationv1.NotificationData) bool {
				return d.HasGoogleChat() && d.GetGoogleChat().GetWebhookUrl() == "https://chat.googleapis.com/x"
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := extractProviderData(context.Background(), tc.model)
			if diags.HasError() {
				t.Fatalf("unexpected diags: %v", diags)
			}
			if !tc.check(got) {
				t.Errorf("wrong oneof arm set; got %v", got.WhichData())
			}
		})
	}
}

func TestExtractProviderData_UnknownProviderErrors(t *testing.T) {
	_, diags := extractProviderData(context.Background(), notificationModel{
		ProviderType: types.StringValue("carrier_pigeon"),
	})
	if !diags.HasError() {
		t.Fatal("expected an error for an unknown provider type")
	}
}

func TestNotificationUpdateRequest_AlwaysSendsUpdateMonitorIDs(t *testing.T) {
	req := newNotificationUpdateRequest("abc", "Test", &notificationv1.NotificationData{}, []string{})

	if !req.HasUpdateMonitorIds() {
		t.Fatal("update_monitor_ids must be present, otherwise the API preserves stale associations")
	}
	if !req.GetUpdateMonitorIds() {
		t.Error("update_monitor_ids = false, want true")
	}
	if req.GetMonitorIds() == nil {
		t.Error("monitor_ids must be set (empty clears associations)")
	}
	if len(req.GetMonitorIds()) != 0 {
		t.Errorf("monitor_ids = %v, want empty", req.GetMonitorIds())
	}
}

func TestNotificationAPIToModel_WhatsAppRoundTrip(t *testing.T) {
	inner := &notificationv1.WhatsappData{}
	inner.SetPhoneNumber("+4915773121555")
	data := &notificationv1.NotificationData{}
	data.SetWhatsapp(inner)

	api := &notificationv1.Notification{}
	api.SetId("1")
	api.SetName("Test")
	api.SetProvider(notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_WHATSAPP)
	api.SetData(data)

	var model notificationModel
	diags := notificationAPIToModel(context.Background(), api, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if model.WhatsApp.IsNull() {
		t.Fatal("WhatsApp list is null; should hold one element")
	}
	obj := model.WhatsApp.Elements()[0].(types.Object)
	if pn := obj.Attributes()["phone_number"].(types.String).ValueString(); pn != "+4915773121555" {
		t.Errorf("phone_number = %q, want +4915773121555", pn)
	}
}

func TestNotificationAPIToModel_MSTeamsRoundTrip(t *testing.T) {
	inner := &notificationv1.MsTeamsData{}
	inner.SetWebhookUrl("https://example.com/teams")
	data := &notificationv1.NotificationData{}
	data.SetMsTeams(inner)

	api := &notificationv1.Notification{}
	api.SetId("5")
	api.SetName("Teams")
	api.SetProvider(notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_MS_TEAMS)
	api.SetData(data)

	var model notificationModel
	diags := notificationAPIToModel(context.Background(), api, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	if model.ProviderType.ValueString() != "ms_teams" {
		t.Errorf("ProviderType = %q, want ms_teams", model.ProviderType.ValueString())
	}
	obj := model.MSTeams.Elements()[0].(types.Object)
	if url := obj.Attributes()["webhook_url"].(types.String).ValueString(); url != "https://example.com/teams" {
		t.Errorf("webhook_url = %q, want https://example.com/teams", url)
	}
}

func TestNotificationAPIToModel_WebhookHeadersRoundTrip(t *testing.T) {
	header := &notificationv1.WebhookHeader{}
	header.SetKey("X-Api-Key")
	header.SetValue("secret")
	inner := &notificationv1.WebhookData{}
	inner.SetEndpoint("https://example.com/hook")
	inner.SetHeaders([]*notificationv1.WebhookHeader{header})
	data := &notificationv1.NotificationData{}
	data.SetWebhook(inner)

	api := &notificationv1.Notification{}
	api.SetId("9")
	api.SetProvider(notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_WEBHOOK)
	api.SetData(data)

	var model notificationModel
	diags := notificationAPIToModel(context.Background(), api, &model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	obj := model.Webhook.Elements()[0].(types.Object)
	headers := obj.Attributes()["headers"].(types.List)
	if len(headers.Elements()) != 1 {
		t.Fatalf("headers length = %d, want 1", len(headers.Elements()))
	}
	h := headers.Elements()[0].(types.Object)
	if k := h.Attributes()["key"].(types.String).ValueString(); k != "X-Api-Key" {
		t.Errorf("header key = %q, want X-Api-Key", k)
	}
}

func TestNotificationAPIToModel_UnknownProviderWarns(t *testing.T) {
	api := &notificationv1.Notification{}
	api.SetId("7")
	api.SetName("Future")
	api.SetProvider(notificationv1.NotificationProvider(99))
	api.SetData(&notificationv1.NotificationData{})

	var model notificationModel
	diags := notificationAPIToModel(context.Background(), api, &model)
	if diags.HasError() {
		t.Fatalf("unexpected error diags: %v", diags)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("warnings = %d, want 1; all diags: %v", diags.WarningsCount(), diags)
	}
	if !strings.Contains(diags.Warnings()[0].Detail(), "99") {
		t.Errorf("warning detail = %q; want it to mention the unknown provider", diags.Warnings()[0].Detail())
	}
}

func TestNotificationAPIToModel_UnknownOpsgenieRegionWarns(t *testing.T) {
	inner := &notificationv1.OpsgenieData{}
	inner.SetApiKey("k")
	inner.SetRegion(notificationv1.OpsgenieRegion(99))
	data := &notificationv1.NotificationData{}
	data.SetOpsgenie(inner)

	api := &notificationv1.Notification{}
	api.SetId("8")
	api.SetName("Opsgenie")
	api.SetProvider(notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_OPSGENIE)
	api.SetData(data)

	var model notificationModel
	diags := notificationAPIToModel(context.Background(), api, &model)
	if diags.HasError() {
		t.Fatalf("unexpected error diags: %v", diags)
	}
	if diags.WarningsCount() != 1 {
		t.Fatalf("warnings = %d, want 1; all diags: %v", diags.WarningsCount(), diags)
	}
	if !strings.Contains(diags.Warnings()[0].Detail(), "99") {
		t.Errorf("warning detail = %q; want it to mention the unknown region", diags.Warnings()[0].Detail())
	}
}

func TestExtractProviderData_NtfyOmitsOptionalFields(t *testing.T) {
	model := notificationModel{
		ProviderType: types.StringValue("ntfy"),
		Ntfy:         ntfyList("alerts", types.StringNull(), types.StringNull()),
	}
	got, diags := extractProviderData(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	ntfy := got.GetNtfy()
	if ntfy.GetTopic() != "alerts" {
		t.Errorf("topic = %q, want alerts", ntfy.GetTopic())
	}
	if ntfy.GetServerUrl() != "" {
		t.Errorf("server_url = %q, want empty when omitted in HCL", ntfy.GetServerUrl())
	}
	if ntfy.HasToken() {
		t.Errorf("token is present despite being null in HCL; got %q", ntfy.GetToken())
	}
}

func TestExtractProviderData_NtfyKeepsOptionalFields(t *testing.T) {
	model := notificationModel{
		ProviderType: types.StringValue("ntfy"),
		Ntfy: ntfyList("alerts",
			types.StringValue("https://ntfy.example.com"),
			types.StringValue("tk_123")),
	}
	got, diags := extractProviderData(context.Background(), model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	ntfy := got.GetNtfy()
	if ntfy.GetServerUrl() != "https://ntfy.example.com" {
		t.Errorf("server_url = %q, want https://ntfy.example.com", ntfy.GetServerUrl())
	}
	if !ntfy.HasToken() {
		t.Fatal("token must be present when set in HCL")
	}
	if ntfy.GetToken() != "tk_123" {
		t.Errorf("token = %q, want tk_123", ntfy.GetToken())
	}
}

func ntfyNotification(topic, serverURL string, token *string) *notificationv1.Notification {
	inner := &notificationv1.NtfyData{}
	inner.SetTopic(topic)
	inner.SetServerUrl(serverURL)
	if token != nil {
		inner.SetToken(*token)
	}
	data := &notificationv1.NotificationData{}
	data.SetNtfy(inner)

	api := &notificationv1.Notification{}
	api.SetId("1128")
	api.SetProvider(notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_NTFY)
	api.SetData(data)
	return api
}

func TestNotificationAPIToModel_NtfyKeepsNullTokenWhenOmittedFromHCL(t *testing.T) {
	// User's HCL has no `token = ...` line. Plan token is null. API doesn't
	// echo token. Post-apply state token MUST be null (not "") — otherwise
	// Terraform aborts with "inconsistent values for sensitive attribute".
	model := notificationModel{
		Ntfy: ntfyList("openstatus", types.StringValue("https://ntfy.sh"), types.StringNull()),
	}

	diags := notificationAPIToModel(context.Background(), ntfyNotification("openstatus", "https://ntfy.sh", nil), &model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	obj := model.Ntfy.Elements()[0].(types.Object)
	tok := obj.Attributes()["token"].(types.String)
	if !tok.IsNull() {
		t.Errorf("token = %q (IsNull=false), want null to match planned null shape", tok.ValueString())
	}
}

func TestNotificationAPIToModel_NtfyPreservesTokenWhenAPIStripsIt(t *testing.T) {
	model := notificationModel{
		Ntfy: ntfyList("alerts",
			types.StringValue("https://ntfy.example.com"),
			types.StringValue("tk_planned")),
	}

	diags := notificationAPIToModel(context.Background(), ntfyNotification("alerts", "https://ntfy.example.com", nil), &model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	obj := model.Ntfy.Elements()[0].(types.Object)
	tok := obj.Attributes()["token"].(types.String)
	if got := tok.ValueString(); got != "tk_planned" {
		t.Errorf("token = %q, want %q (the planned value was lost)", got, "tk_planned")
	}
}

func TestNotificationAPIToModel_NtfyUsesAPITokenIfPresent(t *testing.T) {
	apiToken := "tk_api"
	var model notificationModel
	diags := notificationAPIToModel(context.Background(), ntfyNotification("alerts", "", &apiToken), &model)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags)
	}
	obj := model.Ntfy.Elements()[0].(types.Object)
	tok := obj.Attributes()["token"].(types.String)
	if got := tok.ValueString(); got != "tk_api" {
		t.Errorf("token = %q, want %q", got, "tk_api")
	}
}

func TestProviderTypeMapsCoverEveryGeneratedEnum(t *testing.T) {
	for value, name := range notificationv1.NotificationProvider_name {
		provider := notificationv1.NotificationProvider(value)
		if provider == notificationv1.NotificationProvider_NOTIFICATION_PROVIDER_UNSPECIFIED {
			continue
		}
		if _, ok := providerTypeFromAPI[provider]; !ok {
			t.Errorf("generated provider %s is not mapped to a Terraform value", name)
		}
	}
}

func TestOpsgenieRegionMapsCoverEveryGeneratedEnum(t *testing.T) {
	for value, name := range notificationv1.OpsgenieRegion_name {
		region := notificationv1.OpsgenieRegion(value)
		if region == notificationv1.OpsgenieRegion_OPSGENIE_REGION_UNSPECIFIED {
			continue
		}
		if _, ok := opsgenieRegionFromAPI[region]; !ok {
			t.Errorf("generated Opsgenie region %s is not mapped to a Terraform value", name)
		}
	}
}
