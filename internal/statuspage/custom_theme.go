package statuspage

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	statuspagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Mirrors THEME_VAR_NAMES in openstatus packages/theme-store; the API rejects other names.
var themeVarNames = []string{
	"--radius",
	"--background",
	"--foreground",
	"--card",
	"--card-foreground",
	"--popover",
	"--popover-foreground",
	"--primary",
	"--primary-foreground",
	"--secondary",
	"--secondary-foreground",
	"--muted",
	"--muted-foreground",
	"--accent",
	"--accent-foreground",
	"--border",
	"--input",
	"--ring",
	"--destructive",
	"--chart-1",
	"--chart-2",
	"--chart-3",
	"--chart-4",
	"--chart-5",
	"--success",
	"--warning",
	"--info",
	"--rainbow-1",
	"--rainbow-2",
	"--rainbow-3",
	"--rainbow-4",
	"--rainbow-5",
	"--rainbow-6",
	"--rainbow-7",
	"--rainbow-8",
	"--rainbow-9",
	"--rainbow-10",
	"--rainbow-11",
	"--rainbow-12",
	"--rainbow-13",
	"--rainbow-14",
	"--rainbow-15",
	"--rainbow-16",
	"--rainbow-17",
}

var themeVarNameSet = func() map[string]struct{} {
	s := make(map[string]struct{}, len(themeVarNames))
	for _, n := range themeVarNames {
		s[n] = struct{}{}
	}
	return s
}()

func sortedThemeVarNames() []string {
	out := make([]string, len(themeVarNames))
	copy(out, themeVarNames)
	sort.Strings(out)
	return out
}

const themeVarValueMaxLength = 256

// Values end up in an inline <style> tag; the API bans anything that could break out of it.
var themeVarValuePattern = regexp.MustCompile(`^[^<{};\x00-\x1f]+$`)

var customThemeAttrTypes = map[string]attr.Type{
	"light": types.MapType{ElemType: types.StringType},
	"dark":  types.MapType{ElemType: types.StringType},
}

type customThemeModel struct {
	Light types.Map `tfsdk:"light"`
	Dark  types.Map `tfsdk:"dark"`
}

// Mirrors the server-side validation. Untrimmed values are rejected because
// the server trims on store, which would leave plan and state unequal.
func validateThemeVarEntry(name, value string) []string {
	var errs []string
	if _, ok := themeVarNameSet[name]; !ok {
		errs = append(errs, fmt.Sprintf("unknown CSS variable %q; supported variables: %s", name, strings.Join(sortedThemeVarNames(), ", ")))
		return errs
	}
	if len(value) > themeVarValueMaxLength {
		errs = append(errs, fmt.Sprintf("value of %q must be at most %d characters", name, themeVarValueMaxLength))
	}
	if strings.TrimSpace(value) == "" {
		errs = append(errs, fmt.Sprintf("value of %q must not be empty", name))
	} else {
		if value != strings.TrimSpace(value) {
			errs = append(errs, fmt.Sprintf("value of %q must not have leading or trailing whitespace", name))
		}
		if !themeVarValuePattern.MatchString(value) {
			errs = append(errs, fmt.Sprintf("value of %q contains unsupported characters (`<`, `{`, `}`, `;` and control characters are not allowed)", name))
		}
	}
	return errs
}

var _ validator.Map = (*themeVarsValidator)(nil)

type themeVarsValidator struct{}

func (v themeVarsValidator) Description(ctx context.Context) string {
	return v.MarkdownDescription(ctx)
}

func (v themeVarsValidator) MarkdownDescription(_ context.Context) string {
	return "keys must be supported theme CSS variable names and values must be non-empty, trimmed, at most 256 characters, and free of `<`, `{`, `}`, `;` and control characters"
}

func (v themeVarsValidator) ValidateMap(ctx context.Context, req validator.MapRequest, resp *validator.MapResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	elements := req.ConfigValue.Elements()
	if len(elements) == 0 {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Invalid custom theme",
			"must contain at least one CSS variable; omit the map instead of leaving it empty.",
		)
		return
	}

	for name, elem := range elements {
		strVal, ok := elem.(types.String)
		if !ok || strVal.IsUnknown() {
			continue
		}
		if strVal.IsNull() {
			resp.Diagnostics.AddAttributeError(req.Path.AtMapKey(name), "Invalid custom theme variable",
				fmt.Sprintf("value of %q must not be null.", name))
			continue
		}
		for _, msg := range validateThemeVarEntry(name, strVal.ValueString()) {
			resp.Diagnostics.AddAttributeError(req.Path.AtMapKey(name), "Invalid custom theme variable", msg+".")
		}
	}
}

// Returns nil (field omitted) when the value is null or unknown.
func customThemeToAPI(ctx context.Context, obj types.Object, diags *diag.Diagnostics) *statuspagev1.CustomTheme {
	if obj.IsNull() || obj.IsUnknown() {
		return nil
	}

	var model customThemeModel
	diags.Append(obj.As(ctx, &model, basetypes.ObjectAsOptions{})...)
	if diags.HasError() {
		return nil
	}

	var light, dark map[string]string
	if !model.Light.IsNull() && !model.Light.IsUnknown() {
		diags.Append(model.Light.ElementsAs(ctx, &light, false)...)
	}
	if !model.Dark.IsNull() && !model.Dark.IsUnknown() {
		diags.Append(model.Dark.ElementsAs(ctx, &dark, false)...)
	}
	if diags.HasError() {
		return nil
	}
	// An empty message means "clear" on update; never emit one for a set config value.
	if len(light) == 0 && len(dark) == 0 {
		return nil
	}

	api := &statuspagev1.CustomTheme{}
	api.SetLight(light)
	api.SetDark(dark)
	return api
}

// An absent or empty message maps to a null object.
func customThemeFromAPI(ctx context.Context, api *statuspagev1.CustomTheme, diags *diag.Diagnostics) types.Object {
	if api == nil || (len(api.GetLight()) == 0 && len(api.GetDark()) == 0) {
		return types.ObjectNull(customThemeAttrTypes)
	}

	modeMap := func(vars map[string]string) types.Map {
		if len(vars) == 0 {
			return types.MapNull(types.StringType)
		}
		m, mapDiags := types.MapValueFrom(ctx, types.StringType, vars)
		diags.Append(mapDiags...)
		return m
	}

	obj, objDiags := types.ObjectValue(customThemeAttrTypes, map[string]attr.Value{
		"light": modeMap(api.GetLight()),
		"dark":  modeMap(api.GetDark()),
	})
	diags.Append(objDiags...)
	return obj
}
