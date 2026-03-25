package statuspage

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// --- RPC create request serialization ---

func TestAPIRPCCreateRequest_AllFields(t *testing.T) {
	req := apiRPCCreateRequest{
		Title:            "My Page",
		Slug:             "my-page",
		Description:      "A description",
		HomepageURL:      "https://example.com",
		ContactURL:       "mailto:a@b.com",
		Icon:             "https://example.com/icon.png",
		CustomDomain:     "status.example.com",
		Theme:            "PAGE_THEME_DARK",
		AccessType:       "PAGE_ACCESS_TYPE_PASSWORD_PROTECTED",
		Password:         "secret",
		AuthEmailDomains: []string{"example.com", "test.com"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	expected := map[string]interface{}{
		"title":       "My Page",
		"slug":        "my-page",
		"description": "A description",
		"homepageUrl": "https://example.com",
		"contactUrl":  "mailto:a@b.com",
		"icon":        "https://example.com/icon.png",
		"customDomain": "status.example.com",
		"theme":       "PAGE_THEME_DARK",
		"accessType":  "PAGE_ACCESS_TYPE_PASSWORD_PROTECTED",
		"password":    "secret",
	}
	for k, want := range expected {
		got, ok := raw[k]
		if !ok {
			t.Errorf("field %q missing from JSON", k)
			continue
		}
		if got != want {
			t.Errorf("%s = %v, want %v", k, got, want)
		}
	}

	domains, ok := raw["authEmailDomains"].([]interface{})
	if !ok {
		t.Fatal("authEmailDomains missing or wrong type")
	}
	if len(domains) != 2 || domains[0] != "example.com" || domains[1] != "test.com" {
		t.Errorf("authEmailDomains = %v, want [example.com, test.com]", domains)
	}
}

func TestAPIRPCCreateRequest_OmitsEmptyOptionalFields(t *testing.T) {
	req := apiRPCCreateRequest{
		Title: "My Page",
		Slug:  "my-page",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	omitted := []string{"description", "homepageUrl", "contactUrl", "icon", "customDomain", "theme", "accessType", "password", "authEmailDomains"}
	for _, field := range omitted {
		if _, ok := raw[field]; ok {
			t.Errorf("field %q should be omitted when empty", field)
		}
	}
}

// --- RPC response deserialization ---

func TestAPIRPCStatusPage_ParsesResponse(t *testing.T) {
	apiJSON := `{
		"statusPage": {
			"id": "4409",
			"title": "Traefik Status",
			"slug": "traefik",
			"description": "Traefik services status page",
			"homepageUrl": "https://traefik.io",
			"contactUrl": "mailto:support@traefik.io",
			"icon": "https://traefik.io/favicon.png",
			"customDomain": "",
			"published": false,
			"accessType": "PAGE_ACCESS_TYPE_PUBLIC",
			"password": "",
			"authEmailDomains": [],
			"theme": "PAGE_THEME_SYSTEM",
			"createdAt": "2026-03-24T00:00:00Z",
			"updatedAt": "2026-03-24T00:00:00Z"
		}
	}`

	var resp apiRPCResponse
	if err := json.Unmarshal([]byte(apiJSON), &resp); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	page := resp.StatusPage
	if page.ID != "4409" {
		t.Errorf("ID = %q, want %q", page.ID, "4409")
	}
	if page.Title != "Traefik Status" {
		t.Errorf("Title = %q, want %q", page.Title, "Traefik Status")
	}
	if page.HomepageURL != "https://traefik.io" {
		t.Errorf("HomepageURL = %q, want %q", page.HomepageURL, "https://traefik.io")
	}
	if page.Icon != "https://traefik.io/favicon.png" {
		t.Errorf("Icon = %q, want %q", page.Icon, "https://traefik.io/favicon.png")
	}
	if page.AccessType != "PAGE_ACCESS_TYPE_PUBLIC" {
		t.Errorf("AccessType = %q, want %q", page.AccessType, "PAGE_ACCESS_TYPE_PUBLIC")
	}
	if page.Theme != "PAGE_THEME_SYSTEM" {
		t.Errorf("Theme = %q, want %q", page.Theme, "PAGE_THEME_SYSTEM")
	}
}

// --- enum conversion helpers ---

func TestAccessTypeToProto(t *testing.T) {
	cases := map[string]string{
		"public":       "PAGE_ACCESS_TYPE_PUBLIC",
		"password":     "PAGE_ACCESS_TYPE_PASSWORD_PROTECTED",
		"email-domain": "PAGE_ACCESS_TYPE_AUTHENTICATED",
		"":             "",
		"unknown":      "",
	}
	for tf, want := range cases {
		got := accessTypeToProto(tf)
		if got != want {
			t.Errorf("accessTypeToProto(%q) = %q, want %q", tf, got, want)
		}
	}
}

func TestAccessTypeFromProto(t *testing.T) {
	cases := map[string]string{
		"PAGE_ACCESS_TYPE_PUBLIC":             "public",
		"PAGE_ACCESS_TYPE_PASSWORD_PROTECTED": "password",
		"PAGE_ACCESS_TYPE_AUTHENTICATED":      "email-domain",
		"PAGE_ACCESS_TYPE_UNSPECIFIED":        "public",
		"":                                    "public",
	}
	for proto, want := range cases {
		got := accessTypeFromProto(proto)
		if got != want {
			t.Errorf("accessTypeFromProto(%q) = %q, want %q", proto, got, want)
		}
	}
}

func TestThemeFromProto(t *testing.T) {
	cases := map[string]string{
		"PAGE_THEME_SYSTEM":      "system",
		"PAGE_THEME_LIGHT":       "light",
		"PAGE_THEME_DARK":        "dark",
		"PAGE_THEME_UNSPECIFIED": "system",
		"":                       "system",
	}
	for proto, want := range cases {
		got := themeFromProto(proto)
		if got != want {
			t.Errorf("themeFromProto(%q) = %q, want %q", proto, got, want)
		}
	}
}

// --- rpcAPIToModel ---

func TestRPCAPIToModel_SetsAllFields(t *testing.T) {
	api := apiRPCStatusPage{
		ID:               "42",
		Title:            "Test",
		Slug:             "test",
		Description:      "desc",
		HomepageURL:      "https://example.com",
		ContactURL:       "mailto:a@b.com",
		Icon:             "https://example.com/icon.png",
		CustomDomain:     "status.example.com",
		Published:        true,
		AccessType:       "PAGE_ACCESS_TYPE_PASSWORD_PROTECTED",
		Password:         "secret",
		AuthEmailDomains: []string{"example.com"},
		Theme:            "PAGE_THEME_DARK",
		CreatedAt:        "2026-01-01T00:00:00Z",
		UpdatedAt:        "2026-01-02T00:00:00Z",
	}

	var data statusPageModel
	var diags diag.Diagnostics
	rpcAPIToModel(context.Background(), api, &data, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.ID.ValueString() != "42" {
		t.Errorf("ID = %q, want %q", data.ID.ValueString(), "42")
	}
	if data.Title.ValueString() != "Test" {
		t.Errorf("Title = %q, want %q", data.Title.ValueString(), "Test")
	}
	if data.HomepageURL.ValueString() != "https://example.com" {
		t.Errorf("HomepageURL = %q, want %q", data.HomepageURL.ValueString(), "https://example.com")
	}
	if data.ContactURL.ValueString() != "mailto:a@b.com" {
		t.Errorf("ContactURL = %q, want %q", data.ContactURL.ValueString(), "mailto:a@b.com")
	}
	if data.Icon.ValueString() != "https://example.com/icon.png" {
		t.Errorf("Icon = %q, want %q", data.Icon.ValueString(), "https://example.com/icon.png")
	}
	if data.CustomDomain.ValueString() != "status.example.com" {
		t.Errorf("CustomDomain = %q, want %q", data.CustomDomain.ValueString(), "status.example.com")
	}
	if !data.Published.ValueBool() {
		t.Error("Published = false, want true")
	}
	if data.AccessType.ValueString() != "password" {
		t.Errorf("AccessType = %q, want %q", data.AccessType.ValueString(), "password")
	}
	if data.Password.ValueString() != "secret" {
		t.Errorf("Password = %q, want %q", data.Password.ValueString(), "secret")
	}
	if data.Theme.ValueString() != "dark" {
		t.Errorf("Theme = %q, want %q", data.Theme.ValueString(), "dark")
	}
}

func TestRPCAPIToModel_IconNullWhenAPIReturnsEmptyAndModelNull(t *testing.T) {
	api := apiRPCStatusPage{
		ID:         "1",
		Title:      "Test",
		Slug:       "test",
		AccessType: "PAGE_ACCESS_TYPE_PUBLIC",
		Theme:      "PAGE_THEME_SYSTEM",
	}

	var data statusPageModel
	var diags diag.Diagnostics
	rpcAPIToModel(context.Background(), api, &data, &diags)

	if !data.Icon.IsNull() {
		t.Errorf("Icon should remain null when API returns empty and model is null, got %q", data.Icon.ValueString())
	}
}

func TestRPCAPIToModel_HomepageURLAndContactURLPreservedWhenEmpty(t *testing.T) {
	api := apiRPCStatusPage{
		ID:         "1",
		Title:      "Test",
		Slug:       "test",
		AccessType: "PAGE_ACCESS_TYPE_PUBLIC",
		Theme:      "PAGE_THEME_SYSTEM",
	}

	var data statusPageModel
	var diags diag.Diagnostics
	rpcAPIToModel(context.Background(), api, &data, &diags)

	if !data.HomepageURL.IsNull() {
		t.Errorf("HomepageURL should remain null when API returns empty and model is null, got %q", data.HomepageURL.ValueString())
	}
	if !data.ContactURL.IsNull() {
		t.Errorf("ContactURL should remain null when API returns empty and model is null, got %q", data.ContactURL.ValueString())
	}
}

func TestRPCAPIToModel_AuthEmailDomains(t *testing.T) {
	api := apiRPCStatusPage{
		ID:               "1",
		Title:            "Test",
		Slug:             "test",
		AccessType:       "PAGE_ACCESS_TYPE_AUTHENTICATED",
		AuthEmailDomains: []string{"example.com", "test.com"},
		Theme:            "PAGE_THEME_SYSTEM",
	}

	var data statusPageModel
	var diags diag.Diagnostics
	rpcAPIToModel(context.Background(), api, &data, &diags)

	if data.AuthEmailDomains.IsNull() {
		t.Fatal("AuthEmailDomains should not be null")
	}

	var domains []string
	data.AuthEmailDomains.ElementsAs(context.Background(), &domains, false)
	if len(domains) != 2 || domains[0] != "example.com" || domains[1] != "test.com" {
		t.Errorf("AuthEmailDomains = %v, want [example.com, test.com]", domains)
	}

	if data.AccessType.ValueString() != "email-domain" {
		t.Errorf("AccessType = %q, want %q", data.AccessType.ValueString(), "email-domain")
	}
}
