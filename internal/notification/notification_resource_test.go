package notification

import (
	"context"
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
