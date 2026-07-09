package statuspage

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- validateThemeVarEntry ---

func TestValidateThemeVarEntry_Accepts(t *testing.T) {
	cases := map[string]string{
		"--primary":    "hsl(24 94% 50%)",
		"--radius":     "0.5rem",
		"--rainbow-17": "#ff00aa",
		"--success":    "oklch(0.72 0.19 150)",
	}
	for name, value := range cases {
		if errs := validateThemeVarEntry(name, value); len(errs) != 0 {
			t.Errorf("validateThemeVarEntry(%q, %q) = %v, want no errors", name, value, errs)
		}
	}
}

func TestValidateThemeVarEntry_Rejects(t *testing.T) {
	cases := []struct {
		name    string
		value   string
		wantSub string
	}{
		{"--nope", "red", "unknown CSS variable"},
		{"primary", "red", "unknown CSS variable"},
		{"--primary", "", "must not be empty"},
		{"--primary", "   ", "must not be empty"},
		{"--primary", " red", "leading or trailing whitespace"},
		{"--primary", "red ", "leading or trailing whitespace"},
		{"--primary", "red;}", "unsupported characters"},
		{"--primary", "</style><script>", "unsupported characters"},
		{"--primary", strings.Repeat("a", 257), "at most 256 characters"},
	}
	for _, tc := range cases {
		errs := validateThemeVarEntry(tc.name, tc.value)
		if len(errs) == 0 {
			t.Errorf("validateThemeVarEntry(%q, %q) = no errors, want error containing %q", tc.name, tc.value, tc.wantSub)
			continue
		}
		found := false
		for _, e := range errs {
			if strings.Contains(e, tc.wantSub) {
				found = true
			}
		}
		if !found {
			t.Errorf("validateThemeVarEntry(%q, %q) = %v, want error containing %q", tc.name, tc.value, errs, tc.wantSub)
		}
	}
}

// --- themeVarsValidator ---

func runThemeVarsValidator(t *testing.T, value types.Map) diag.Diagnostics {
	t.Helper()
	req := validator.MapRequest{
		Path:        path.Root("custom_theme").AtName("light"),
		ConfigValue: value,
	}
	resp := &validator.MapResponse{}
	themeVarsValidator{}.ValidateMap(context.Background(), req, resp)
	return resp.Diagnostics
}

func TestThemeVarsValidator_ValidMap(t *testing.T) {
	m, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"--primary": "hsl(24 94% 50%)",
		"--success": "green",
	})
	if diags := runThemeVarsValidator(t, m); diags.HasError() {
		t.Errorf("unexpected diagnostics: %v", diags)
	}
}

func TestThemeVarsValidator_NullAndUnknownPass(t *testing.T) {
	if diags := runThemeVarsValidator(t, types.MapNull(types.StringType)); diags.HasError() {
		t.Errorf("null map: unexpected diagnostics: %v", diags)
	}
	if diags := runThemeVarsValidator(t, types.MapUnknown(types.StringType)); diags.HasError() {
		t.Errorf("unknown map: unexpected diagnostics: %v", diags)
	}
}

func TestThemeVarsValidator_EmptyMapRejected(t *testing.T) {
	m, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{})
	if diags := runThemeVarsValidator(t, m); !diags.HasError() {
		t.Error("empty map should be rejected")
	}
}

func TestThemeVarsValidator_UnknownVarRejected(t *testing.T) {
	m, _ := types.MapValueFrom(context.Background(), types.StringType, map[string]string{
		"--not-a-var": "red",
	})
	if diags := runThemeVarsValidator(t, m); !diags.HasError() {
		t.Error("unknown variable name should be rejected")
	}
}

// --- conversions ---

func customThemeObject(t *testing.T, light, dark map[string]string) types.Object {
	t.Helper()
	ctx := context.Background()
	modeVal := func(vars map[string]string) attr.Value {
		if vars == nil {
			return types.MapNull(types.StringType)
		}
		m, diags := types.MapValueFrom(ctx, types.StringType, vars)
		if diags.HasError() {
			t.Fatalf("unexpected diagnostics: %v", diags)
		}
		return m
	}
	obj, diags := types.ObjectValue(customThemeAttrTypes, map[string]attr.Value{
		"light": modeVal(light),
		"dark":  modeVal(dark),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	return obj
}

func TestCustomThemeToAPI_NullAndUnknown(t *testing.T) {
	var diags diag.Diagnostics
	if got := customThemeToAPI(context.Background(), types.ObjectNull(customThemeAttrTypes), &diags); got != nil {
		t.Errorf("null object = %v, want nil", got)
	}
	if got := customThemeToAPI(context.Background(), types.ObjectUnknown(customThemeAttrTypes), &diags); got != nil {
		t.Errorf("unknown object = %v, want nil", got)
	}
	if diags.HasError() {
		t.Errorf("unexpected diagnostics: %v", diags)
	}
}

func TestCustomThemeToAPI_LightOnly(t *testing.T) {
	obj := customThemeObject(t, map[string]string{"--primary": "red"}, nil)
	var diags diag.Diagnostics
	api := customThemeToAPI(context.Background(), obj, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if api == nil {
		t.Fatal("api = nil, want value")
	}
	if api.Light["--primary"] != "red" {
		t.Errorf("Light = %v, want {--primary: red}", api.Light)
	}
	if api.Dark != nil {
		t.Errorf("Dark = %v, want nil", api.Dark)
	}
}

func TestCustomThemeFromAPI_RoundTrip(t *testing.T) {
	in := &apiCustomTheme{
		Light: map[string]string{"--primary": "hsl(24 94% 50%)"},
		Dark:  map[string]string{"--primary": "hsl(24 94% 60%)", "--background": "black"},
	}
	var diags diag.Diagnostics
	obj := customThemeFromAPI(context.Background(), in, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	out := customThemeToAPI(context.Background(), obj, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if out.Light["--primary"] != in.Light["--primary"] {
		t.Errorf("Light = %v, want %v", out.Light, in.Light)
	}
	if len(out.Dark) != 2 || out.Dark["--background"] != "black" {
		t.Errorf("Dark = %v, want %v", out.Dark, in.Dark)
	}
}

func TestCustomThemeFromAPI_NilAndEmptyAreNull(t *testing.T) {
	var diags diag.Diagnostics
	if got := customThemeFromAPI(context.Background(), nil, &diags); !got.IsNull() {
		t.Errorf("nil api = %v, want null object", got)
	}
	if got := customThemeFromAPI(context.Background(), &apiCustomTheme{}, &diags); !got.IsNull() {
		t.Errorf("empty api = %v, want null object", got)
	}
	if got := customThemeFromAPI(context.Background(), &apiCustomTheme{Light: map[string]string{}, Dark: map[string]string{}}, &diags); !got.IsNull() {
		t.Errorf("empty maps = %v, want null object", got)
	}
	if diags.HasError() {
		t.Errorf("unexpected diagnostics: %v", diags)
	}
}

func TestCustomThemeFromAPI_UnsetModeIsNull(t *testing.T) {
	var diags diag.Diagnostics
	obj := customThemeFromAPI(context.Background(), &apiCustomTheme{
		Light: map[string]string{"--primary": "red"},
	}, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	dark := obj.Attributes()["dark"].(types.Map)
	if !dark.IsNull() {
		t.Errorf("dark = %v, want null", dark)
	}
}

// --- wire serialization ---

func TestAPIStatusPageCreateRequest_CustomTheme(t *testing.T) {
	req := apiStatusPageCreateRequest{
		Title: "Test",
		Slug:  "test",
		CustomTheme: &apiCustomTheme{
			Light: map[string]string{"--primary": "red"},
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	theme, ok := raw["customTheme"].(map[string]interface{})
	if !ok {
		t.Fatalf("customTheme = %v, want object", raw["customTheme"])
	}
	light, ok := theme["light"].(map[string]interface{})
	if !ok || light["--primary"] != "red" {
		t.Errorf("customTheme.light = %v, want {--primary: red}", theme["light"])
	}
	if _, ok := theme["dark"]; ok {
		t.Errorf("customTheme.dark should be omitted when empty, got %v", theme["dark"])
	}
}

func TestAPIStatusPageCreateRequest_OmitsNilCustomTheme(t *testing.T) {
	req := apiStatusPageCreateRequest{Title: "Test", Slug: "test"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["customTheme"]; ok {
		t.Errorf("customTheme should be omitted when nil, got %v", raw["customTheme"])
	}
}

func TestAPIStatusPageUpdateRequest_EmptyCustomThemeClears(t *testing.T) {
	// The API treats an omitted customTheme as "keep" and an empty message as "clear".
	req := apiStatusPageUpdateRequest{ID: "42", CustomTheme: &apiCustomTheme{}}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	theme, ok := raw["customTheme"].(map[string]interface{})
	if !ok {
		t.Fatal("customTheme should be present as an empty object to clear it")
	}
	if len(theme) != 0 {
		t.Errorf("customTheme = %v, want empty object", theme)
	}
}

func TestAPIStatusPageUpdateRequest_OmitsNilCustomTheme(t *testing.T) {
	req := apiStatusPageUpdateRequest{ID: "42"}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if _, ok := raw["customTheme"]; ok {
		t.Errorf("customTheme should be omitted when nil, got %v", raw["customTheme"])
	}
}

// --- model mapping ---

func TestStatusPageAPIToModel_CustomTheme(t *testing.T) {
	api := apiStatusPage{
		ID:    "1",
		Title: "X",
		Slug:  "x",
		CustomTheme: &apiCustomTheme{
			Dark: map[string]string{"--background": "black"},
		},
	}
	var data statusPageModel
	var diags diag.Diagnostics
	statusPageAPIToModel(context.Background(), api, &data, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if data.CustomTheme.IsNull() {
		t.Fatal("CustomTheme should not be null")
	}
	dark := data.CustomTheme.Attributes()["dark"].(types.Map)
	var vars map[string]string
	dark.ElementsAs(context.Background(), &vars, false)
	if vars["--background"] != "black" {
		t.Errorf("dark = %v, want {--background: black}", vars)
	}
}

func TestStatusPageAPIToModel_CustomThemeNullWhenAbsent(t *testing.T) {
	api := apiStatusPage{ID: "1", Title: "X", Slug: "x"}
	var data statusPageModel
	var diags diag.Diagnostics
	statusPageAPIToModel(context.Background(), api, &data, &diags)
	if !data.CustomTheme.IsNull() {
		t.Errorf("CustomTheme = %v, want null", data.CustomTheme)
	}
}

func TestAPIStatusPage_ParsesCustomThemeResponse(t *testing.T) {
	apiJSON := `{
		"statusPage": {
			"id": "1",
			"title": "X",
			"slug": "x",
			"customTheme": {
				"light": {"--primary": "hsl(24 94% 50%)"},
				"dark": {"--primary": "hsl(24 94% 60%)"}
			}
		}
	}`
	var resp apiStatusPageResponse
	if err := json.Unmarshal([]byte(apiJSON), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	theme := resp.StatusPage.CustomTheme
	if theme == nil {
		t.Fatal("CustomTheme = nil, want value")
	}
	if theme.Light["--primary"] != "hsl(24 94% 50%)" || theme.Dark["--primary"] != "hsl(24 94% 60%)" {
		t.Errorf("CustomTheme = %+v, want light/dark --primary set", theme)
	}
}
