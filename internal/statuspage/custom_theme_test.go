package statuspage

import (
	"context"
	"strings"
	"testing"

	statuspagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"

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

func TestThemeVarsValidator_NullValueRejected(t *testing.T) {
	m, diags := types.MapValue(types.StringType, map[string]attr.Value{
		"--primary": types.StringNull(),
	})
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if diags := runThemeVarsValidator(t, m); !diags.HasError() {
		t.Error("null element value should be rejected")
	}
}

func TestCustomThemeToAPI_AllModesNullIsNil(t *testing.T) {
	obj := customThemeObject(t, nil, nil)
	var diags diag.Diagnostics
	if got := customThemeToAPI(context.Background(), obj, &diags); got != nil {
		t.Errorf("both modes null = %v, want nil", got)
	}
	if diags.HasError() {
		t.Errorf("unexpected diagnostics: %v", diags)
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
	if api.GetLight()["--primary"] != "red" {
		t.Errorf("Light = %v, want {--primary: red}", api.GetLight())
	}
	if api.GetDark() != nil {
		t.Errorf("Dark = %v, want nil", api.GetDark())
	}
}

func customTheme(light, dark map[string]string) *statuspagev1.CustomTheme {
	theme := &statuspagev1.CustomTheme{}
	theme.SetLight(light)
	theme.SetDark(dark)
	return theme
}

func TestCustomThemeFromAPI_RoundTrip(t *testing.T) {
	in := customTheme(
		map[string]string{"--primary": "hsl(24 94% 50%)"},
		map[string]string{"--primary": "hsl(24 94% 60%)", "--background": "black"},
	)
	var diags diag.Diagnostics
	obj := customThemeFromAPI(context.Background(), in, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	out := customThemeToAPI(context.Background(), obj, &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if out.GetLight()["--primary"] != in.GetLight()["--primary"] {
		t.Errorf("Light = %v, want %v", out.GetLight(), in.GetLight())
	}
	if len(out.GetDark()) != 2 || out.GetDark()["--background"] != "black" {
		t.Errorf("Dark = %v, want %v", out.GetDark(), in.GetDark())
	}
}

func TestCustomThemeFromAPI_NilAndEmptyAreNull(t *testing.T) {
	var diags diag.Diagnostics
	if got := customThemeFromAPI(context.Background(), nil, &diags); !got.IsNull() {
		t.Errorf("nil api = %v, want null object", got)
	}
	if got := customThemeFromAPI(context.Background(), &statuspagev1.CustomTheme{}, &diags); !got.IsNull() {
		t.Errorf("empty api = %v, want null object", got)
	}
	if got := customThemeFromAPI(context.Background(), customTheme(map[string]string{}, map[string]string{}), &diags); !got.IsNull() {
		t.Errorf("empty maps = %v, want null object", got)
	}
	if diags.HasError() {
		t.Errorf("unexpected diagnostics: %v", diags)
	}
}

func TestCustomThemeFromAPI_UnsetModeIsNull(t *testing.T) {
	var diags diag.Diagnostics
	obj := customThemeFromAPI(context.Background(), customTheme(map[string]string{"--primary": "red"}, nil), &diags)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	dark := obj.Attributes()["dark"].(types.Map)
	if !dark.IsNull() {
		t.Errorf("dark = %v, want null", dark)
	}
}

func TestCreateStatusPageRequest_CustomThemePresence(t *testing.T) {
	req := &statuspagev1.CreateStatusPageRequest{}
	req.SetTitle("Test")
	req.SetSlug("test")
	if req.HasCustomTheme() {
		t.Error("custom_theme must be absent until set")
	}

	req.SetCustomTheme(customTheme(map[string]string{"--primary": "red"}, nil))
	if !req.HasCustomTheme() {
		t.Fatal("custom_theme must be present once set")
	}
	if req.GetCustomTheme().GetLight()["--primary"] != "red" {
		t.Errorf("light = %v, want {--primary: red}", req.GetCustomTheme().GetLight())
	}
	if len(req.GetCustomTheme().GetDark()) != 0 {
		t.Errorf("dark = %v, want empty", req.GetCustomTheme().GetDark())
	}
}

// The API treats an omitted custom_theme as "keep" and an empty message as
// "clear", so presence is what distinguishes the two.
func TestUpdateStatusPageRequest_EmptyCustomThemeClears(t *testing.T) {
	keep := &statuspagev1.UpdateStatusPageRequest{}
	keep.SetId("42")
	if keep.HasCustomTheme() {
		t.Error("custom_theme must be absent when not set, otherwise every update clears the theme")
	}

	clear := &statuspagev1.UpdateStatusPageRequest{}
	clear.SetId("42")
	clear.SetCustomTheme(&statuspagev1.CustomTheme{})
	if !clear.HasCustomTheme() {
		t.Fatal("an empty custom_theme must still be present in order to clear")
	}
	if len(clear.GetCustomTheme().GetLight()) != 0 || len(clear.GetCustomTheme().GetDark()) != 0 {
		t.Error("clearing custom_theme must send an empty message")
	}
}

func TestStatusPageAPIToModel_CustomTheme(t *testing.T) {
	api := &statuspagev1.StatusPage{}
	api.SetId("1")
	api.SetTitle("X")
	api.SetSlug("x")
	api.SetCustomTheme(customTheme(nil, map[string]string{"--background": "black"}))

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
	api := &statuspagev1.StatusPage{}
	api.SetId("1")
	api.SetTitle("X")
	api.SetSlug("x")

	var data statusPageModel
	var diags diag.Diagnostics
	statusPageAPIToModel(context.Background(), api, &data, &diags)
	if !data.CustomTheme.IsNull() {
		t.Errorf("CustomTheme = %v, want null", data.CustomTheme)
	}
}
