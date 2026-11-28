// Package playbook manages OpenStatus playbooks: markdown guidance the
// OpenStatus agent reads when it runs incidents, writes status page updates,
// schedules maintenance or drafts postmortems.
package playbook

import (
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unicode/utf16"

	"terraform-provider-openstatus/internal/client"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type providerConfig = client.ProviderConfig

// Limits mirror the OpenStatus API so mistakes fail at plan time instead of
// halfway through an apply.
const (
	maxSlugChars          = 64
	maxTitleChars         = 100
	maxDescriptionChars   = 500
	maxContentChars       = 10_000
	maxVariableValueChars = 500
)

// BuiltinSlugs come with a default title, description and template. They are
// written with UpdatePlaybook (CreatePlaybook refuses them), and deleting one
// makes the agent fall back to the OpenStatus default.
var BuiltinSlugs = []string{
	"incident-response",
	"status-page",
	"maintenance-windows",
	"postmortem",
}

func IsBuiltinSlug(slug string) bool {
	return slices.Contains(BuiltinSlugs, slug)
}

var (
	slugPattern         = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	variableNamePattern = regexp.MustCompile(`^[a-z0-9_]{1,64}$`)
	// The API only substitutes `{{name}}`, optionally padded with whitespace.
	variableRefPattern = regexp.MustCompile(`\{\{\s*([a-z0-9_]{1,64})\s*\}\}`)
)

// jsLen counts UTF-16 code units, which is how the API (JavaScript) measures
// string length.
func jsLen(s string) int {
	return len(utf16.Encode([]rune(s)))
}

// VariableNames returns the variables referenced in content, each once.
func VariableNames(content string) []string {
	seen := map[string]bool{}
	var names []string
	for _, m := range variableRefPattern.FindAllStringSubmatch(content, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}
	return names
}

// render fills in content the way the API does, to enforce its length limit
// on the filled-in text.
func render(content string, variables map[string]string) string {
	return variableRefPattern.ReplaceAllStringFunc(content, func(match string) string {
		name := variableRefPattern.FindStringSubmatch(match)[1]
		if v := strings.TrimSpace(variables[name]); v != "" {
			return v
		}
		return match
	})
}

// validateText checks a value the API trims before measuring.
func validateText(p path.Path, label, value string, maxChars int) diag.Diagnostics {
	var diags diag.Diagnostics
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		diags.AddAttributeError(p, "Invalid "+label, label+" cannot be empty or only whitespace.")
	} else if n := jsLen(trimmed); n > maxChars {
		diags.AddAttributeError(p, "Invalid "+label,
			fmt.Sprintf("%s is limited to %d characters, got %d.", label, maxChars, n))
	}
	return diags
}

// validateContent checks the content and its variables. The API silently
// drops variables the content does not reference and variables whose value is
// blank; both would leave Terraform with an inconsistent result after apply,
// so they are rejected here instead.
func validateContent(content string, variables map[string]string, variablesKnown bool) diag.Diagnostics {
	diags := validateText(path.Root("content"), "Content", content, maxContentChars)
	if diags.HasError() || !variablesKnown {
		return diags
	}

	used := map[string]bool{}
	for _, name := range VariableNames(content) {
		used[name] = true
	}

	names := make([]string, 0, len(variables))
	for name := range variables {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		p := path.Root("variables").AtMapKey(name)
		value := strings.TrimSpace(variables[name])
		switch {
		case !variableNamePattern.MatchString(name):
			diags.AddAttributeError(p, "Invalid variable name",
				fmt.Sprintf("Variable %q must use only lowercase letters, digits and underscores (max 64 characters).", name))
		case !used[name]:
			diags.AddAttributeError(p, "Unused variable",
				fmt.Sprintf("Variable %q is not referenced in content. Add {{%s}} to the content or remove the variable; OpenStatus drops values the content does not use.", name, name))
		case value == "":
			diags.AddAttributeError(p, "Empty variable",
				fmt.Sprintf("Variable %q has an empty value. Remove it instead; OpenStatus does not store empty values.", name))
		case jsLen(value) > maxVariableValueChars:
			diags.AddAttributeError(p, "Variable value too long",
				fmt.Sprintf("Variable %q is limited to %d characters, got %d.", name, maxVariableValueChars, jsLen(value)))
		}
	}
	if diags.HasError() {
		return diags
	}

	if n := jsLen(render(strings.TrimSpace(content), variables)); n > maxContentChars {
		diags.AddAttributeError(path.Root("content"), "Content too long",
			fmt.Sprintf("A playbook is limited to %d characters with its variables filled in, got %d.", maxContentChars, n))
	}
	return diags
}

// keepIfTrimEqual returns the configured value when the API only trimmed it,
// so a heredoc's trailing newline doesn't read as drift or fail the apply.
func keepIfTrimEqual(prior types.String, api string) types.String {
	if !prior.IsNull() && !prior.IsUnknown() && strings.TrimSpace(prior.ValueString()) == api {
		return prior
	}
	return types.StringValue(api)
}

// keepVariablesIfTrimEqual applies keepIfTrimEqual to each variable value.
func keepVariablesIfTrimEqual(prior map[string]string, api map[string]string) map[string]string {
	out := make(map[string]string, len(api))
	for k, v := range api {
		if p, ok := prior[k]; ok && strings.TrimSpace(p) == v {
			out[k] = p
		} else {
			out[k] = v
		}
	}
	return out
}
