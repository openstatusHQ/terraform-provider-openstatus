package statuspage

import (
	"context"
	"testing"

	statuspagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- create request construction ---

func TestCreateStatusPageRequest_AllFields(t *testing.T) {
	req := &statuspagev1.CreateStatusPageRequest{}
	req.SetTitle("Acme Corp Status")
	req.SetSlug("acme-corp")
	req.SetDescription("Acme Corp services status")
	req.SetHomepageUrl("https://example.com")
	req.SetContactUrl("mailto:a@b.com")
	req.SetIcon("https://example.com/icon.png")
	req.SetCustomDomain("status.example.com")
	req.SetTheme(themeToProto("dark"))
	req.SetAccessType(accessTypeToProto("password"))
	req.SetPassword("secret")
	req.SetAuthEmailDomains([]string{"example.com", "test.com"})

	if req.GetTitle() != "Acme Corp Status" {
		t.Errorf("title = %q, want Acme Corp Status", req.GetTitle())
	}
	if req.GetTheme() != statuspagev1.PageTheme_PAGE_THEME_DARK {
		t.Errorf("theme = %v, want PAGE_THEME_DARK", req.GetTheme())
	}
	if req.GetAccessType() != statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PASSWORD_PROTECTED {
		t.Errorf("access_type = %v, want PAGE_ACCESS_TYPE_PASSWORD_PROTECTED", req.GetAccessType())
	}
	if req.GetPassword() != "secret" {
		t.Errorf("password = %q, want secret", req.GetPassword())
	}
	domains := req.GetAuthEmailDomains()
	if len(domains) != 2 || domains[0] != "example.com" || domains[1] != "test.com" {
		t.Errorf("auth_email_domains = %v, want [example.com test.com]", domains)
	}
}

func TestCreateStatusPageRequest_OptionalFieldsAbsentWhenUnset(t *testing.T) {
	req := &statuspagev1.CreateStatusPageRequest{}
	req.SetTitle("Acme Corp Status")
	req.SetSlug("acme-corp")

	for name, has := range map[string]bool{
		"description":       req.HasDescription(),
		"homepage_url":      req.HasHomepageUrl(),
		"contact_url":       req.HasContactUrl(),
		"icon":              req.HasIcon(),
		"custom_domain":     req.HasCustomDomain(),
		"theme":             req.HasTheme(),
		"access_type":       req.HasAccessType(),
		"password":          req.HasPassword(),
		"allowed_ip_ranges": req.HasAllowedIpRanges(),
		"default_locale":    req.HasDefaultLocale(),
		"allow_index":       req.HasAllowIndex(),
		"custom_theme":      req.HasCustomTheme(),
	} {
		if has {
			t.Errorf("%s should be absent when unset", name)
		}
	}
	if len(req.GetAuthEmailDomains()) != 0 {
		t.Error("auth_email_domains should be empty when unset")
	}
	if len(req.GetLocales()) != 0 {
		t.Error("locales should be empty when unset")
	}
}

func TestCreateStatusPageRequest_H3Fields(t *testing.T) {
	req := &statuspagev1.CreateStatusPageRequest{}
	req.SetTitle("Test")
	req.SetSlug("test")
	req.SetTheme(themeToProto("dark"))
	req.SetDefaultLocale(localeToProto("fr"))
	req.SetLocales([]statuspagev1.Locale{localeToProto("en"), localeToProto("fr")})
	req.SetAllowIndex(false)
	req.SetAllowedIpRanges("10.0.0.0/8")

	if req.GetDefaultLocale() != statuspagev1.Locale_LOCALE_FR {
		t.Errorf("default_locale = %v, want LOCALE_FR", req.GetDefaultLocale())
	}
	if len(req.GetLocales()) != 2 {
		t.Errorf("locales = %v, want two entries", req.GetLocales())
	}
	if !req.HasAllowIndex() {
		t.Error("allow_index must be present even when false")
	}
	if req.GetAllowIndex() {
		t.Error("allow_index = true, want false")
	}
	if req.GetAllowedIpRanges() != "10.0.0.0/8" {
		t.Errorf("allowed_ip_ranges = %q, want 10.0.0.0/8", req.GetAllowedIpRanges())
	}
}

// --- update request presence ---

func TestUpdateStatusPageRequest_OmitsUnsetFields(t *testing.T) {
	req := &statuspagev1.UpdateStatusPageRequest{}
	req.SetId("42")
	req.SetTitle("Acme Corp Status")

	for name, has := range map[string]bool{
		"slug":              req.HasSlug(),
		"description":       req.HasDescription(),
		"homepage_url":      req.HasHomepageUrl(),
		"contact_url":       req.HasContactUrl(),
		"icon":              req.HasIcon(),
		"custom_domain":     req.HasCustomDomain(),
		"access_type":       req.HasAccessType(),
		"password":          req.HasPassword(),
		"theme":             req.HasTheme(),
		"default_locale":    req.HasDefaultLocale(),
		"allow_index":       req.HasAllowIndex(),
		"allowed_ip_ranges": req.HasAllowedIpRanges(),
	} {
		if has {
			t.Errorf("%s should be absent when unset, otherwise update clobbers it", name)
		}
	}
}

// An empty string must still be *present* so the server clears the field.
func TestUpdateStatusPageRequest_EmptyStringClearsField(t *testing.T) {
	req := &statuspagev1.UpdateStatusPageRequest{}
	req.SetId("42")
	req.SetContactUrl("")
	req.SetHomepageUrl("")

	if !req.HasContactUrl() {
		t.Error("contact_url should be present with an empty string to clear it")
	}
	if !req.HasHomepageUrl() {
		t.Error("homepage_url should be present with an empty string to clear it")
	}
	if req.GetContactUrl() != "" || req.GetHomepageUrl() != "" {
		t.Error("cleared fields should carry empty strings")
	}
}

func TestUpdateStatusPageRequest_EmptySliceClearsAuthEmailDomains(t *testing.T) {
	req := &statuspagev1.UpdateStatusPageRequest{}
	req.SetId("42")
	req.SetAuthEmailDomains([]string{})

	if req.GetAuthEmailDomains() == nil {
		t.Fatal("auth_email_domains should be set (empty) to clear it")
	}
	if len(req.GetAuthEmailDomains()) != 0 {
		t.Errorf("auth_email_domains = %v, want empty", req.GetAuthEmailDomains())
	}
}

// --- enum conversion helpers ---

func TestAccessTypeToProto(t *testing.T) {
	cases := map[string]statuspagev1.PageAccessType{
		"public":       statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PUBLIC,
		"password":     statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PASSWORD_PROTECTED,
		"email-domain": statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_AUTHENTICATED,
		"ip":           statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_IP_RESTRICTED,
		"":             statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_UNSPECIFIED,
		"unknown":      statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_UNSPECIFIED,
	}
	for tf, want := range cases {
		if got := accessTypeToProto(tf); got != want {
			t.Errorf("accessTypeToProto(%q) = %v, want %v", tf, got, want)
		}
	}
}

func TestAccessTypeFromProto(t *testing.T) {
	knownCases := map[statuspagev1.PageAccessType]string{
		statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PUBLIC:             "public",
		statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PASSWORD_PROTECTED: "password",
		statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_AUTHENTICATED:      "email-domain",
		statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_IP_RESTRICTED:      "ip",
	}
	for proto, want := range knownCases {
		got, ok := accessTypeFromProto(proto)
		if !ok {
			t.Errorf("accessTypeFromProto(%v) ok=false, want true", proto)
			continue
		}
		if got != want {
			t.Errorf("accessTypeFromProto(%v) = %q, want %q", proto, got, want)
		}
	}
	for _, proto := range []statuspagev1.PageAccessType{
		statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_UNSPECIFIED,
		statuspagev1.PageAccessType(99),
	} {
		if _, ok := accessTypeFromProto(proto); ok {
			t.Errorf("accessTypeFromProto(%v) ok=true, want false", proto)
		}
	}
}

func TestThemeRoundTrip(t *testing.T) {
	cases := map[string]statuspagev1.PageTheme{
		"system": statuspagev1.PageTheme_PAGE_THEME_SYSTEM,
		"light":  statuspagev1.PageTheme_PAGE_THEME_LIGHT,
		"dark":   statuspagev1.PageTheme_PAGE_THEME_DARK,
	}
	for tf, proto := range cases {
		if got := themeToProto(tf); got != proto {
			t.Errorf("themeToProto(%q) = %v, want %v", tf, got, proto)
		}
		got, ok := themeFromProto(proto)
		if !ok || got != tf {
			t.Errorf("themeFromProto(%v) = (%q, %v), want (%q, true)", proto, got, ok, tf)
		}
	}
	for _, proto := range []statuspagev1.PageTheme{
		statuspagev1.PageTheme_PAGE_THEME_UNSPECIFIED,
		statuspagev1.PageTheme(99),
	} {
		if _, ok := themeFromProto(proto); ok {
			t.Errorf("themeFromProto(%v) ok=true, want false", proto)
		}
	}
}

func TestLocaleRoundTrip(t *testing.T) {
	cases := map[string]statuspagev1.Locale{
		"en": statuspagev1.Locale_LOCALE_EN,
		"fr": statuspagev1.Locale_LOCALE_FR,
		"de": statuspagev1.Locale_LOCALE_DE,
	}
	for tf, proto := range cases {
		if got := localeToProto(tf); got != proto {
			t.Errorf("localeToProto(%q) = %v, want %v", tf, got, proto)
		}
		got, ok := localeFromProto(proto)
		if !ok || got != tf {
			t.Errorf("localeFromProto(%v) = (%q, %v), want (%q, true)", proto, got, ok, tf)
		}
	}
	for _, proto := range []statuspagev1.Locale{
		statuspagev1.Locale_LOCALE_UNSPECIFIED,
		statuspagev1.Locale(99),
	} {
		if _, ok := localeFromProto(proto); ok {
			t.Errorf("localeFromProto(%v) ok=true, want false", proto)
		}
	}
}

// The API stores locales as a set: a duplicated locale comes back once, which
// used to fail apply with "Provider produced inconsistent result after apply".
// Duplicates must be rejected at plan time instead.
func TestLocalesSchemaRejectsDuplicates(t *testing.T) {
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	(&statusPageResource{}).Schema(ctx, resource.SchemaRequest{}, schemaResp)
	localesAttr, ok := schemaResp.Schema.Attributes["locales"].(schema.ListAttribute)
	if !ok {
		t.Fatalf("locales attribute = %T, want schema.ListAttribute", schemaResp.Schema.Attributes["locales"])
	}

	cases := map[string]struct {
		locales []string
		wantErr bool
	}{
		"unique":             {locales: []string{"de", "en", "fr"}},
		"duplicate":          {locales: []string{"de", "en", "en", "en", "fr"}, wantErr: true},
		"duplicate adjacent": {locales: []string{"en", "en"}, wantErr: true},
		"unknown locale":     {locales: []string{"es"}, wantErr: true},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			elems := make([]attr.Value, 0, len(tc.locales))
			for _, l := range tc.locales {
				elems = append(elems, types.StringValue(l))
			}
			list, diags := types.ListValue(types.StringType, elems)
			if diags.HasError() {
				t.Fatalf("building list: %v", diags)
			}

			var got diag.Diagnostics
			for _, v := range localesAttr.Validators {
				vResp := &validator.ListResponse{}
				v.ValidateList(ctx, validator.ListRequest{
					Path:        path.Root("locales"),
					ConfigValue: list,
				}, vResp)
				got.Append(vResp.Diagnostics...)
			}

			if got.HasError() != tc.wantErr {
				t.Errorf("locales %v: error = %v, want %v (%v)", tc.locales, got.HasError(), tc.wantErr, got)
			}
		})
	}
}

func TestEnumMapsCoverEveryGeneratedValue(t *testing.T) {
	for value, name := range statuspagev1.PageAccessType_name {
		access := statuspagev1.PageAccessType(value)
		if access == statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_UNSPECIFIED {
			continue
		}
		if _, ok := accessTypeFromProto(access); !ok {
			t.Errorf("generated access type %s is not mapped to a Terraform value", name)
		}
	}
	for value, name := range statuspagev1.PageTheme_name {
		theme := statuspagev1.PageTheme(value)
		if theme == statuspagev1.PageTheme_PAGE_THEME_UNSPECIFIED {
			continue
		}
		if _, ok := themeFromProto(theme); !ok {
			t.Errorf("generated theme %s is not mapped to a Terraform value", name)
		}
	}
	for value, name := range statuspagev1.Locale_name {
		locale := statuspagev1.Locale(value)
		if locale == statuspagev1.Locale_LOCALE_UNSPECIFIED {
			continue
		}
		if _, ok := localeFromProto(locale); !ok {
			t.Errorf("generated locale %s is not mapped to a Terraform value", name)
		}
	}
}

// --- statusPageAPIToModel ---

func statusPage(id, title, slug string) *statuspagev1.StatusPage {
	p := &statuspagev1.StatusPage{}
	p.SetId(id)
	p.SetTitle(title)
	p.SetSlug(slug)
	return p
}

func TestStatusPageAPIToModel_SetsAllFields(t *testing.T) {
	api := statusPage("42", "Acme Corp Status", "acme-corp")
	api.SetDescription("Acme Corp services status")
	api.SetHomepageUrl("https://acme.example.com")
	api.SetContactUrl("mailto:support@acme.example.com")
	api.SetIcon("https://acme.example.com/icon.png")
	api.SetCustomDomain("status.acme.example.com")
	api.SetPublished(true)
	api.SetAccessType(statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PASSWORD_PROTECTED)
	api.SetPassword("secret")
	api.SetAuthEmailDomains([]string{"example.com"})
	api.SetTheme(statuspagev1.PageTheme_PAGE_THEME_DARK)
	api.SetCreatedAt("2026-01-01T00:00:00Z")
	api.SetUpdatedAt("2026-01-02T00:00:00Z")

	var data statusPageModel
	var diags diag.Diagnostics
	statusPageAPIToModel(context.Background(), api, &data, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	checks := map[string]struct{ got, want string }{
		"ID":           {data.ID.ValueString(), "42"},
		"Title":        {data.Title.ValueString(), "Acme Corp Status"},
		"HomepageURL":  {data.HomepageURL.ValueString(), "https://acme.example.com"},
		"ContactURL":   {data.ContactURL.ValueString(), "mailto:support@acme.example.com"},
		"Icon":         {data.Icon.ValueString(), "https://acme.example.com/icon.png"},
		"CustomDomain": {data.CustomDomain.ValueString(), "status.acme.example.com"},
		"AccessType":   {data.AccessType.ValueString(), "password"},
		"Password":     {data.Password.ValueString(), "secret"},
		"Theme":        {data.Theme.ValueString(), "dark"},
	}
	for name, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", name, c.got, c.want)
		}
	}
	if !data.Published.ValueBool() {
		t.Error("Published = false, want true")
	}
}

func TestStatusPageAPIToModel_EmptyOptionalsStayNull(t *testing.T) {
	api := statusPage("1", "Acme Minimal", "acme-minimal")
	api.SetAccessType(statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_PUBLIC)
	api.SetTheme(statuspagev1.PageTheme_PAGE_THEME_SYSTEM)

	var data statusPageModel
	var diags diag.Diagnostics
	statusPageAPIToModel(context.Background(), api, &data, &diags)

	if !data.Icon.IsNull() {
		t.Errorf("Icon should remain null when API returns empty and model is null, got %q", data.Icon.ValueString())
	}
	if !data.HomepageURL.IsNull() {
		t.Errorf("HomepageURL should remain null, got %q", data.HomepageURL.ValueString())
	}
	if !data.ContactURL.IsNull() {
		t.Errorf("ContactURL should remain null, got %q", data.ContactURL.ValueString())
	}
}

func TestStatusPageAPIToModel_AuthEmailDomains(t *testing.T) {
	api := statusPage("1", "Acme Secure", "acme-secure")
	api.SetAccessType(statuspagev1.PageAccessType_PAGE_ACCESS_TYPE_AUTHENTICATED)
	api.SetAuthEmailDomains([]string{"example.com", "test.com"})
	api.SetTheme(statuspagev1.PageTheme_PAGE_THEME_SYSTEM)

	var data statusPageModel
	var diags diag.Diagnostics
	statusPageAPIToModel(context.Background(), api, &data, &diags)

	if data.AuthEmailDomains.IsNull() {
		t.Fatal("AuthEmailDomains should not be null")
	}
	var domains []string
	data.AuthEmailDomains.ElementsAs(context.Background(), &domains, false)
	if len(domains) != 2 || domains[0] != "example.com" || domains[1] != "test.com" {
		t.Errorf("AuthEmailDomains = %v, want [example.com test.com]", domains)
	}
	if data.AccessType.ValueString() != "email-domain" {
		t.Errorf("AccessType = %q, want email-domain", data.AccessType.ValueString())
	}
}

func TestStatusPageAPIToModel_UnknownLocaleWarns(t *testing.T) {
	api := statusPage("1", "X", "x")
	api.SetLocales([]statuspagev1.Locale{statuspagev1.Locale_LOCALE_EN, statuspagev1.Locale(99)})

	var data statusPageModel
	var diags diag.Diagnostics
	statusPageAPIToModel(context.Background(), api, &data, &diags)
	if diags.WarningsCount() != 1 {
		t.Fatalf("warnings = %d, want 1", diags.WarningsCount())
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
