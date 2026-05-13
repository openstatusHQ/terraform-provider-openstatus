package statuspage

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// --- RPC create request serialization ---

func TestAPIStatusPageCreateRequest_AllFields(t *testing.T) {
	req := apiStatusPageCreateRequest{
		Title:            "Acme Corp Status",
		Slug:             "acme-corp",
		Description:      "Acme Corp services status",
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
		"title":        "Acme Corp Status",
		"slug":         "acme-corp",
		"description":  "Acme Corp services status",
		"homepageUrl":  "https://example.com",
		"contactUrl":   "mailto:a@b.com",
		"icon":         "https://example.com/icon.png",
		"customDomain": "status.example.com",
		"theme":        "PAGE_THEME_DARK",
		"accessType":   "PAGE_ACCESS_TYPE_PASSWORD_PROTECTED",
		"password":     "secret",
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

func TestAPIStatusPageCreateRequest_OmitsEmptyOptionalFields(t *testing.T) {
	req := apiStatusPageCreateRequest{
		Title: "Acme Corp Status",
		Slug:  "acme-corp",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	omitted := []string{"description", "homepageUrl", "contactUrl", "icon", "customDomain", "theme", "accessType", "password", "authEmailDomains", "allowedIpRanges", "defaultLocale", "locales", "allowIndex"}
	for _, field := range omitted {
		if _, ok := raw[field]; ok {
			t.Errorf("field %q should be omitted when empty", field)
		}
	}
}

// --- RPC response deserialization ---

func TestAPIStatusPage_ParsesResponse(t *testing.T) {
	apiJSON := `{
		"statusPage": {
			"id": "4409",
			"title": "Acme Corp Status",
			"slug": "acme-corp",
			"description": "Acme Corp services status page",
			"homepageUrl": "https://acme.example.com",
			"contactUrl": "mailto:support@acme.example.com",
			"icon": "https://acme.example.com/favicon.png",
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

	var resp apiStatusPageResponse
	if err := json.Unmarshal([]byte(apiJSON), &resp); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	page := resp.StatusPage
	if page.ID != "4409" {
		t.Errorf("ID = %q, want %q", page.ID, "4409")
	}
	if page.Title != "Acme Corp Status" {
		t.Errorf("Title = %q, want %q", page.Title, "Acme Corp Status")
	}
	if page.HomepageURL != "https://acme.example.com" {
		t.Errorf("HomepageURL = %q, want %q", page.HomepageURL, "https://acme.example.com")
	}
	if page.Icon != "https://acme.example.com/favicon.png" {
		t.Errorf("Icon = %q, want %q", page.Icon, "https://acme.example.com/favicon.png")
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
	knownCases := map[string]string{
		"PAGE_ACCESS_TYPE_PUBLIC":             "public",
		"PAGE_ACCESS_TYPE_PASSWORD_PROTECTED": "password",
		"PAGE_ACCESS_TYPE_AUTHENTICATED":      "email-domain",
		"PAGE_ACCESS_TYPE_IP_RESTRICTED":      "ip",
	}
	for proto, want := range knownCases {
		got, ok := accessTypeFromProto(proto)
		if !ok {
			t.Errorf("accessTypeFromProto(%q) ok=false, want true", proto)
			continue
		}
		if got != want {
			t.Errorf("accessTypeFromProto(%q) = %q, want %q", proto, got, want)
		}
	}
	for _, proto := range []string{"PAGE_ACCESS_TYPE_UNSPECIFIED", "", "FUTURE"} {
		if _, ok := accessTypeFromProto(proto); ok {
			t.Errorf("accessTypeFromProto(%q) ok=true, want false", proto)
		}
	}
}

func TestThemeFromProto(t *testing.T) {
	knownCases := map[string]string{
		"PAGE_THEME_SYSTEM": "system",
		"PAGE_THEME_LIGHT":  "light",
		"PAGE_THEME_DARK":   "dark",
	}
	for proto, want := range knownCases {
		got, ok := themeFromProto(proto)
		if !ok {
			t.Errorf("themeFromProto(%q) ok=false, want true", proto)
			continue
		}
		if got != want {
			t.Errorf("themeFromProto(%q) = %q, want %q", proto, got, want)
		}
	}
	for _, proto := range []string{"PAGE_THEME_UNSPECIFIED", "", "NEON"} {
		if _, ok := themeFromProto(proto); ok {
			t.Errorf("themeFromProto(%q) ok=true, want false", proto)
		}
	}
}

// --- statusPageAPIToModel ---

func TestStatusPageAPIToModel_SetsAllFields(t *testing.T) {
	api := apiStatusPage{
		ID:               "42",
		Title:            "Acme Corp Status",
		Slug:             "acme-corp",
		Description:      "Acme Corp services status",
		HomepageURL:      "https://acme.example.com",
		ContactURL:       "mailto:support@acme.example.com",
		Icon:             "https://acme.example.com/icon.png",
		CustomDomain:     "status.acme.example.com",
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
	statusPageAPIToModel(context.Background(), api, &data, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.ID.ValueString() != "42" {
		t.Errorf("ID = %q, want %q", data.ID.ValueString(), "42")
	}
	if data.Title.ValueString() != "Acme Corp Status" {
		t.Errorf("Title = %q, want %q", data.Title.ValueString(), "Acme Corp Status")
	}
	if data.HomepageURL.ValueString() != "https://acme.example.com" {
		t.Errorf("HomepageURL = %q, want %q", data.HomepageURL.ValueString(), "https://acme.example.com")
	}
	if data.ContactURL.ValueString() != "mailto:support@acme.example.com" {
		t.Errorf("ContactURL = %q, want %q", data.ContactURL.ValueString(), "mailto:support@acme.example.com")
	}
	if data.Icon.ValueString() != "https://acme.example.com/icon.png" {
		t.Errorf("Icon = %q, want %q", data.Icon.ValueString(), "https://acme.example.com/icon.png")
	}
	if data.CustomDomain.ValueString() != "status.acme.example.com" {
		t.Errorf("CustomDomain = %q, want %q", data.CustomDomain.ValueString(), "status.acme.example.com")
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

func TestStatusPageAPIToModel_IconNullWhenAPIReturnsEmptyAndModelNull(t *testing.T) {
	api := apiStatusPage{
		ID:         "1",
		Title:      "Acme Minimal",
		Slug:       "acme-minimal",
		AccessType: "PAGE_ACCESS_TYPE_PUBLIC",
		Theme:      "PAGE_THEME_SYSTEM",
	}

	var data statusPageModel
	var diags diag.Diagnostics
	statusPageAPIToModel(context.Background(), api, &data, &diags)

	if !data.Icon.IsNull() {
		t.Errorf("Icon should remain null when API returns empty and model is null, got %q", data.Icon.ValueString())
	}
}

func TestStatusPageAPIToModel_HomepageURLAndContactURLPreservedWhenEmpty(t *testing.T) {
	api := apiStatusPage{
		ID:         "1",
		Title:      "Acme Minimal",
		Slug:       "acme-minimal",
		AccessType: "PAGE_ACCESS_TYPE_PUBLIC",
		Theme:      "PAGE_THEME_SYSTEM",
	}

	var data statusPageModel
	var diags diag.Diagnostics
	statusPageAPIToModel(context.Background(), api, &data, &diags)

	if !data.HomepageURL.IsNull() {
		t.Errorf("HomepageURL should remain null when API returns empty and model is null, got %q", data.HomepageURL.ValueString())
	}
	if !data.ContactURL.IsNull() {
		t.Errorf("ContactURL should remain null when API returns empty and model is null, got %q", data.ContactURL.ValueString())
	}
}

func TestStatusPageAPIToModel_AuthEmailDomains(t *testing.T) {
	api := apiStatusPage{
		ID:               "1",
		Title:            "Acme Secure",
		Slug:             "acme-secure",
		AccessType:       "PAGE_ACCESS_TYPE_AUTHENTICATED",
		AuthEmailDomains: []string{"example.com", "test.com"},
		Theme:            "PAGE_THEME_SYSTEM",
	}

	var data statusPageModel
	var diags diag.Diagnostics
	statusPageAPIToModel(context.Background(), api, &data, &diags)

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

// --- Update request serialization ---

func TestAPIStatusPageUpdateRequest_AllFields(t *testing.T) {
	title := "Acme Corp Status"
	slug := "acme-corp"
	desc := "Updated description"
	homepage := "https://acme.example.com"
	contact := "mailto:support@acme.example.com"
	icon := "https://acme.example.com/icon.png"
	domain := "status.acme.example.com"
	access := "PAGE_ACCESS_TYPE_PASSWORD_PROTECTED"
	pass := "secret"

	req := apiStatusPageUpdateRequest{
		ID:               "42",
		Title:            &title,
		Slug:             &slug,
		Description:      &desc,
		HomepageURL:      &homepage,
		ContactURL:       &contact,
		Icon:             &icon,
		CustomDomain:     &domain,
		AccessType:       &access,
		Password:         &pass,
		AuthEmailDomains: &[]string{"acme.example.com"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if raw["id"] != "42" {
		t.Errorf("id = %v, want 42", raw["id"])
	}
	if raw["title"] != "Acme Corp Status" {
		t.Errorf("title = %v, want Acme Corp Status", raw["title"])
	}
	if raw["password"] != "secret" {
		t.Errorf("password = %v, want secret", raw["password"])
	}
	domains, ok := raw["authEmailDomains"].([]interface{})
	if !ok || len(domains) != 1 || domains[0] != "acme.example.com" {
		t.Errorf("authEmailDomains = %v, want [acme.example.com]", raw["authEmailDomains"])
	}
}

func TestAPIStatusPageUpdateRequest_OmitsNilFields(t *testing.T) {
	title := "Acme Corp Status"
	req := apiStatusPageUpdateRequest{
		ID:    "42",
		Title: &title,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	omitted := []string{"slug", "description", "homepageUrl", "contactUrl", "icon", "customDomain", "accessType", "password", "authEmailDomains"}
	for _, field := range omitted {
		if _, ok := raw[field]; ok {
			t.Errorf("field %q should be omitted when nil", field)
		}
	}
}

func TestAPIStatusPageUpdateRequest_EmptyStringClearsField(t *testing.T) {
	empty := ""
	req := apiStatusPageUpdateRequest{
		ID:          "42",
		ContactURL:  &empty,
		HomepageURL: &empty,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	// Empty strings should be present (not omitted) to clear the field server-side.
	if _, ok := raw["contactUrl"]; !ok {
		t.Error("contactUrl should be present with empty string to clear it")
	}
	if _, ok := raw["homepageUrl"]; !ok {
		t.Error("homepageUrl should be present with empty string to clear it")
	}
}

func TestAPIStatusPageUpdateRequest_EmptySliceClearsAuthEmailDomains(t *testing.T) {
	domains := []string{}
	req := apiStatusPageUpdateRequest{
		ID:               "42",
		AuthEmailDomains: &domains,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	val, ok := raw["authEmailDomains"]
	if !ok {
		t.Fatal("authEmailDomains should be present with empty slice to clear it")
	}
	arr, ok := val.([]interface{})
	if !ok || len(arr) != 0 {
		t.Errorf("authEmailDomains = %v, want empty array", val)
	}
}

func TestAccessTypeToProto_IP(t *testing.T) {
	if got := accessTypeToProto("ip"); got != "PAGE_ACCESS_TYPE_IP_RESTRICTED" {
		t.Errorf("accessTypeToProto(\"ip\") = %q, want PAGE_ACCESS_TYPE_IP_RESTRICTED", got)
	}
}

func TestLocaleRoundTrip(t *testing.T) {
	cases := map[string]string{"en": "LOCALE_EN", "fr": "LOCALE_FR", "de": "LOCALE_DE"}
	for tf, proto := range cases {
		if got := localeToProto(tf); got != proto {
			t.Errorf("localeToProto(%q) = %q, want %q", tf, got, proto)
		}
		got, ok := localeFromProto(proto)
		if !ok || got != tf {
			t.Errorf("localeFromProto(%q) = (%q, %v), want (%q, true)", proto, got, ok, tf)
		}
	}
	for _, proto := range []string{"LOCALE_UNSPECIFIED", "", "LOCALE_ES"} {
		if _, ok := localeFromProto(proto); ok {
			t.Errorf("localeFromProto(%q) ok=true, want false", proto)
		}
	}
}

func TestThemeToProto_All(t *testing.T) {
	cases := map[string]string{"system": "PAGE_THEME_SYSTEM", "light": "PAGE_THEME_LIGHT", "dark": "PAGE_THEME_DARK"}
	for tf, proto := range cases {
		if got := themeToProto(tf); got != proto {
			t.Errorf("themeToProto(%q) = %q, want %q", tf, got, proto)
		}
	}
}

func TestSlugPattern_Accepts(t *testing.T) {
	for _, s := range []string{"acme", "acme-corp", "a-b-c", "abc123", "123-456"} {
		if !slugPattern.MatchString(s) {
			t.Errorf("slugPattern.MatchString(%q) = false, want true", s)
		}
	}
}

func TestSlugPattern_Rejects(t *testing.T) {
	for _, s := range []string{"Acme", "acme_corp", "-acme", "acme-", "acme--corp", "", "acme corp"} {
		if slugPattern.MatchString(s) {
			t.Errorf("slugPattern.MatchString(%q) = true, want false", s)
		}
	}
}

func TestAPIStatusPageCreateRequest_NewH3Fields(t *testing.T) {
	allowIndex := false
	req := apiStatusPageCreateRequest{
		Title:           "Test",
		Slug:            "test",
		Theme:           "PAGE_THEME_DARK",
		DefaultLocale:   "LOCALE_FR",
		Locales:         []string{"LOCALE_EN", "LOCALE_FR"},
		AllowIndex:      &allowIndex,
		AllowedIPRanges: "10.0.0.0/8",
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	expect := map[string]interface{}{
		"theme":           "PAGE_THEME_DARK",
		"defaultLocale":   "LOCALE_FR",
		"allowedIpRanges": "10.0.0.0/8",
		"allowIndex":      false,
	}
	for k, want := range expect {
		got, ok := raw[k]
		if !ok {
			t.Errorf("field %q missing", k)
			continue
		}
		if got != want {
			t.Errorf("%s = %v, want %v", k, got, want)
		}
	}
	locales, ok := raw["locales"].([]interface{})
	if !ok || len(locales) != 2 {
		t.Errorf("locales = %v, want [LOCALE_EN, LOCALE_FR]", raw["locales"])
	}
}

func TestAPIStatusPageUpdateRequest_NilFieldsAbsent(t *testing.T) {
	req := apiStatusPageUpdateRequest{ID: "42"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	absent := []string{"theme", "defaultLocale", "locales", "allowIndex", "allowedIpRanges"}
	for _, f := range absent {
		if _, ok := raw[f]; ok {
			t.Errorf("field %q should be absent when nil pointer; got %v", f, raw[f])
		}
	}
}

func TestStatusPageAPIToModel_UnknownLocaleWarns(t *testing.T) {
	api := apiStatusPage{
		ID:      "1",
		Title:   "X",
		Slug:    "x",
		Locales: []string{"LOCALE_EN", "LOCALE_ES"},
	}
	var data statusPageModel
	var diags diag.Diagnostics
	statusPageAPIToModel(context.Background(), api, &data, &diags)
	if diags.WarningsCount() != 1 {
		t.Fatalf("warnings = %d, want 1", diags.WarningsCount())
	}
}
