package statuspage

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- RPC create request serialization ---

func TestAPIRPCCreateRequest_AllFields(t *testing.T) {
	req := apiRPCCreateRequest{
		Title:       "My Page",
		Slug:        "my-page",
		Description: "A description",
		HomepageURL: "https://example.com",
		ContactURL:  "mailto:a@b.com",
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

	omitted := []string{"description", "homepageUrl", "contactUrl"}
	for _, field := range omitted {
		if _, ok := raw[field]; ok {
			t.Errorf("field %q should be omitted when empty", field)
		}
	}
}

// --- REST update request serialization ---

func TestAPIRESTUpdateRequest_AllFields(t *testing.T) {
	req := apiRESTUpdateRequest{
		Icon:             "https://example.com/icon.png",
		CustomDomain:     "status.example.com",
		AccessType:       "password",
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
		"icon":         "https://example.com/icon.png",
		"customDomain": "status.example.com",
		"accessType":   "password",
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

func TestAPIRESTUpdateRequest_OmitsEmptyOptionalFields(t *testing.T) {
	req := apiRESTUpdateRequest{}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("unexpected marshal error: %v", err)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	omitted := []string{"icon", "customDomain", "accessType", "password", "authEmailDomains"}
	for _, field := range omitted {
		if _, ok := raw[field]; ok {
			t.Errorf("field %q should be omitted when empty", field)
		}
	}
}

// --- REST response deserialization ---

func TestAPIRESTStatusPage_ParsesResponse(t *testing.T) {
	apiJSON := `{
		"id": 4409,
		"title": "Traefik Status",
		"slug": "traefik",
		"description": "Traefik services status page",
		"icon": "https://traefik.io/favicon.png",
		"customDomain": "",
		"published": false,
		"accessType": "public",
		"password": "",
		"authEmailDomains": [],
		"theme": "",
		"createdAt": "2026-03-24T00:00:00Z",
		"updatedAt": "2026-03-24T00:00:00Z"
	}`

	var page apiRESTStatusPage
	if err := json.Unmarshal([]byte(apiJSON), &page); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	if page.ID != 4409 {
		t.Errorf("ID = %d, want 4409", page.ID)
	}
	if page.Title != "Traefik Status" {
		t.Errorf("Title = %q, want %q", page.Title, "Traefik Status")
	}
	if page.Icon != "https://traefik.io/favicon.png" {
		t.Errorf("Icon = %q, want %q", page.Icon, "https://traefik.io/favicon.png")
	}
	if page.AccessType != "public" {
		t.Errorf("AccessType = %q, want %q", page.AccessType, "public")
	}
}

// --- restAPIToModel ---

func TestRestAPIToModel_SetsAllFields(t *testing.T) {
	api := apiRESTStatusPage{
		ID:               42,
		Title:            "Test",
		Slug:             "test",
		Description:      "desc",
		Icon:             "https://example.com/icon.png",
		CustomDomain:     "status.example.com",
		Published:        true,
		AccessType:       "password",
		Password:         "secret",
		AuthEmailDomains: []string{"example.com"},
		Theme:            "dark",
		CreatedAt:        "2026-01-01T00:00:00Z",
		UpdatedAt:        "2026-01-02T00:00:00Z",
	}

	var data statusPageModel
	var diags diag.Diagnostics
	restAPIToModel(context.Background(), api, &data, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	if data.ID.ValueString() != "42" {
		t.Errorf("ID = %q, want %q", data.ID.ValueString(), "42")
	}
	if data.Title.ValueString() != "Test" {
		t.Errorf("Title = %q, want %q", data.Title.ValueString(), "Test")
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
}

func TestRestAPIToModel_PreservesHomepageURLAndContactURL(t *testing.T) {
	api := apiRESTStatusPage{
		ID:         1,
		Title:      "Test",
		Slug:       "test",
		AccessType: "public",
	}

	data := statusPageModel{
		HomepageURL: types.StringValue("https://traefik.io"),
		ContactURL:  types.StringValue("mailto:support@traefik.io"),
	}

	var diags diag.Diagnostics
	restAPIToModel(context.Background(), api, &data, &diags)

	if data.HomepageURL.ValueString() != "https://traefik.io" {
		t.Errorf("HomepageURL = %q, want %q (should be preserved)", data.HomepageURL.ValueString(), "https://traefik.io")
	}
	if data.ContactURL.ValueString() != "mailto:support@traefik.io" {
		t.Errorf("ContactURL = %q, want %q (should be preserved)", data.ContactURL.ValueString(), "mailto:support@traefik.io")
	}
}

func TestRestAPIToModel_IconNullWhenAPIReturnsEmptyAndModelNull(t *testing.T) {
	api := apiRESTStatusPage{
		ID:         1,
		Title:      "Test",
		Slug:       "test",
		AccessType: "public",
	}

	var data statusPageModel
	var diags diag.Diagnostics
	restAPIToModel(context.Background(), api, &data, &diags)

	if !data.Icon.IsNull() {
		t.Errorf("Icon should remain null when API returns empty and model is null, got %q", data.Icon.ValueString())
	}
}

func TestRestAPIToModel_AuthEmailDomains(t *testing.T) {
	api := apiRESTStatusPage{
		ID:               1,
		Title:            "Test",
		Slug:             "test",
		AccessType:       "email-domain",
		AuthEmailDomains: []string{"example.com", "test.com"},
	}

	var data statusPageModel
	var diags diag.Diagnostics
	restAPIToModel(context.Background(), api, &data, &diags)

	if data.AuthEmailDomains.IsNull() {
		t.Fatal("AuthEmailDomains should not be null")
	}

	var domains []string
	data.AuthEmailDomains.ElementsAs(context.Background(), &domains, false)
	if len(domains) != 2 || domains[0] != "example.com" || domains[1] != "test.com" {
		t.Errorf("AuthEmailDomains = %v, want [example.com, test.com]", domains)
	}
}

// --- buildRESTUpdateRequest ---

func TestBuildRESTUpdateRequest_ReturnsNilWhenNoRESTFields(t *testing.T) {
	data := &statusPageModel{
		Title:            types.StringValue("My Page"),
		Slug:             types.StringValue("my-page"),
		HomepageURL:      types.StringValue("https://example.com"),
		ContactURL:       types.StringValue("mailto:a@b.com"),
		AuthEmailDomains: types.ListNull(types.StringType),
	}

	var diags diag.Diagnostics
	req := buildRESTUpdateRequest(data, context.Background(), &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if req != nil {
		t.Error("expected nil when no REST-only fields are set")
	}
}

func TestBuildRESTUpdateRequest_ReturnsRequestWhenIconSet(t *testing.T) {
	data := &statusPageModel{
		Icon:             types.StringValue("https://example.com/icon.png"),
		AuthEmailDomains: types.ListNull(types.StringType),
	}

	var diags diag.Diagnostics
	req := buildRESTUpdateRequest(data, context.Background(), &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if req == nil {
		t.Fatal("expected non-nil request when icon is set")
	}
	if req.Icon != "https://example.com/icon.png" {
		t.Errorf("Icon = %q, want %q", req.Icon, "https://example.com/icon.png")
	}
}

func TestBuildRESTUpdateRequest_AllRESTFields(t *testing.T) {
	domains, _ := types.ListValueFrom(context.Background(), types.StringType, []string{"example.com"})
	data := &statusPageModel{
		Icon:             types.StringValue("https://example.com/icon.png"),
		CustomDomain:     types.StringValue("status.example.com"),
		AccessType:       types.StringValue("email-domain"),
		Password:         types.StringValue("secret"),
		AuthEmailDomains: domains,
	}

	var diags diag.Diagnostics
	req := buildRESTUpdateRequest(data, context.Background(), &diags)

	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if req == nil {
		t.Fatal("expected non-nil request")
	}
	if req.Icon != "https://example.com/icon.png" {
		t.Errorf("Icon = %q, want %q", req.Icon, "https://example.com/icon.png")
	}
	if req.CustomDomain != "status.example.com" {
		t.Errorf("CustomDomain = %q, want %q", req.CustomDomain, "status.example.com")
	}
	if req.AccessType != "email-domain" {
		t.Errorf("AccessType = %q, want %q", req.AccessType, "email-domain")
	}
	if req.Password != "secret" {
		t.Errorf("Password = %q, want %q", req.Password, "secret")
	}
	if len(req.AuthEmailDomains) != 1 || req.AuthEmailDomains[0] != "example.com" {
		t.Errorf("AuthEmailDomains = %v, want [example.com]", req.AuthEmailDomains)
	}
}
