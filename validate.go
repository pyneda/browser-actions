package browseractions

import (
	"fmt"
	"net/url"
	"strings"
)

type ValidationProfile string

const (
	ValidationProfileStrict       ValidationProfile = "strict"
	ValidationProfileLegacySukyan ValidationProfile = "sukyan-legacy"
)

func ValidateScript(script BrowserActions, profile ValidationProfile) error {
	var diagnostics []Diagnostic

	if strings.TrimSpace(script.Title) == "" {
		diagnostics = append(diagnostics, Diagnostic{
			Kind:    "validation",
			Path:    "title",
			Code:    "required",
			Message: "title is required",
		})
	}
	if len(script.Actions) == 0 {
		diagnostics = append(diagnostics, Diagnostic{
			Kind:    "validation",
			Path:    "actions",
			Code:    "min_items",
			Message: "at least one action is required",
		})
	}

	if err := ValidateActions(script.Actions, profile); err != nil {
		if de, ok := AsDiagnosticsError(err); ok {
			diagnostics = append(diagnostics, de.Diagnostics...)
		} else {
			return err
		}
	}

	if len(diagnostics) > 0 {
		return diagnosticsErr("validation", nil, diagnostics...)
	}
	return nil
}

func ValidateActions(actions []Action, profile ValidationProfile) error {
	if profile == "" {
		profile = ValidationProfileStrict
	}

	var diagnostics []Diagnostic
	for i, action := range actions {
		basePath := fmt.Sprintf("actions.%d", i)
		switch profile {
		case ValidationProfileLegacySukyan:
			diagnostics = append(diagnostics, validateActionLegacy(basePath, action)...)
		case ValidationProfileStrict:
			diagnostics = append(diagnostics, validateActionStrict(basePath, action)...)
		default:
			diagnostics = append(diagnostics, Diagnostic{
				Kind:    "validation",
				Path:    "profile",
				Code:    "invalid",
				Message: fmt.Sprintf("unsupported validation profile %q", profile),
			})
		}
	}

	if len(diagnostics) > 0 {
		return diagnosticsErr("validation", nil, diagnostics...)
	}
	return nil
}

func validateActionLegacy(basePath string, action Action) []Diagnostic {
	var diags []Diagnostic
	if !isValidActionType(action.Type) {
		return append(diags, vdiag(basePath+".type", "oneof", "type must be a supported action"))
	}

	switch action.Type {
	case ActionNavigate:
		if strings.TrimSpace(action.URL) == "" {
			diags = append(diags, vdiag(basePath+".url", "required", "url is required for navigate"))
		} else if !isProbablyURL(action.URL) {
			diags = append(diags, vdiag(basePath+".url", "url", "url must be valid"))
		}
	case ActionClick:
		if strings.TrimSpace(action.Selector) == "" {
			diags = append(diags, vdiag(basePath+".selector", "required", "selector is required for click"))
		}
	case ActionFill:
		if strings.TrimSpace(action.Selector) == "" {
			diags = append(diags, vdiag(basePath+".selector", "required", "selector is required for fill"))
		}
		if action.Value == "" {
			diags = append(diags, vdiag(basePath+".value", "required", "value is required for fill"))
		}
	case ActionWait:
		if action.For != "" && !isValidWaitCondition(action.For) {
			diags = append(diags, vdiag(basePath+".for", "oneof", "wait condition must be visible/hidden/enabled/load"))
		}
	case ActionAssert:
		if !isValidAssertCondition(action.Condition) {
			diags = append(diags, vdiag(basePath+".condition", "required", "condition is required for assert"))
		}
	case ActionScroll:
		if !isValidScrollPosition(action.Position) {
			diags = append(diags, vdiag(basePath+".position", "required", "position is required for scroll"))
		}
	case ActionScreenshot:
		if strings.TrimSpace(action.File) == "" {
			diags = append(diags, vdiag(basePath+".file", "required", "file is required for screenshot in legacy profile"))
		}
	case ActionSleep:
		if action.Duration <= 0 {
			diags = append(diags, vdiag(basePath+".duration", "gt", "duration must be greater than 0"))
		}
	case ActionEvaluate:
		if strings.TrimSpace(action.Expression) == "" {
			diags = append(diags, vdiag(basePath+".expression", "required", "expression is required for evaluate"))
		}
	}

	return diags
}

func validateActionStrict(basePath string, action Action) []Diagnostic {
	var diags []Diagnostic
	if !isValidActionType(action.Type) {
		return append(diags, vdiag(basePath+".type", "oneof", "type must be a supported action"))
	}

	switch action.Type {
	case ActionNavigate:
		if strings.TrimSpace(action.URL) == "" {
			diags = append(diags, vdiag(basePath+".url", "required", "url is required for navigate"))
		} else if !isProbablyURL(action.URL) {
			diags = append(diags, vdiag(basePath+".url", "url", "url must be valid"))
		}
	case ActionClick:
		if strings.TrimSpace(action.Selector) == "" {
			diags = append(diags, vdiag(basePath+".selector", "required", "selector is required for click"))
		}
	case ActionFill:
		if strings.TrimSpace(action.Selector) == "" {
			diags = append(diags, vdiag(basePath+".selector", "required", "selector is required for fill"))
		}
		if action.Value == "" {
			diags = append(diags, vdiag(basePath+".value", "required", "value is required for fill"))
		}
	case ActionWait:
		if !isValidWaitCondition(action.For) {
			diags = append(diags, vdiag(basePath+".for", "required", "wait condition is required"))
			break
		}
		if action.For != WaitLoad && strings.TrimSpace(action.Selector) == "" {
			diags = append(diags, vdiag(basePath+".selector", "required", "selector is required unless waiting for page load"))
		}
	case ActionAssert:
		if strings.TrimSpace(action.Selector) == "" {
			diags = append(diags, vdiag(basePath+".selector", "required", "selector is required for assert"))
		}
		if !isValidAssertCondition(action.Condition) {
			diags = append(diags, vdiag(basePath+".condition", "required", "assert condition is required"))
			break
		}
		if (action.Condition == AssertContains || action.Condition == AssertEquals) && action.Value == "" {
			diags = append(diags, vdiag(basePath+".value", "required", "value is required for contains/equals assertions"))
		}
	case ActionScroll:
		if !isValidScrollPosition(action.Position) {
			diags = append(diags, vdiag(basePath+".position", "required", "scroll position is required"))
		}
	case ActionScreenshot:
		// file is optional in strict mode because callers may want inline-only screenshots.
	case ActionSleep:
		if action.Duration <= 0 {
			diags = append(diags, vdiag(basePath+".duration", "gt", "duration must be greater than 0"))
		}
	case ActionEvaluate:
		if strings.TrimSpace(action.Expression) == "" {
			diags = append(diags, vdiag(basePath+".expression", "required", "expression is required for evaluate"))
		}
	}

	if action.For != "" && !isValidWaitCondition(action.For) {
		diags = append(diags, vdiag(basePath+".for", "oneof", "wait condition must be visible/hidden/enabled/load"))
	}
	if action.Condition != "" && !isValidAssertCondition(action.Condition) {
		diags = append(diags, vdiag(basePath+".condition", "oneof", "assert condition must be contains/equals/visible/hidden"))
	}
	if action.Position != "" && !isValidScrollPosition(action.Position) {
		diags = append(diags, vdiag(basePath+".position", "oneof", "scroll position must be top/bottom"))
	}

	return dedupeDiagnostics(diags)
}

func vdiag(path, code, msg string) Diagnostic {
	return Diagnostic{
		Kind:    "validation",
		Path:    path,
		Code:    code,
		Message: msg,
	}
}

func dedupeDiagnostics(in []Diagnostic) []Diagnostic {
	seen := make(map[string]struct{}, len(in))
	out := make([]Diagnostic, 0, len(in))
	for _, d := range in {
		k := d.Path + "|" + d.Code + "|" + d.Message
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, d)
	}
	return out
}

func isProbablyURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil {
		return false
	}
	return u.Scheme != "" && u.Host != ""
}

func isValidActionType(t ActionType) bool {
	switch t {
	case ActionNavigate, ActionClick, ActionFill, ActionWait, ActionAssert, ActionScroll, ActionScreenshot, ActionSleep, ActionEvaluate:
		return true
	default:
		return false
	}
}

func isValidWaitCondition(c WaitCondition) bool {
	switch c {
	case WaitVisible, WaitHidden, WaitEnabled, WaitLoad:
		return true
	default:
		return false
	}
}

func isValidAssertCondition(c AssertCondition) bool {
	switch c {
	case AssertContains, AssertEquals, AssertVisible, AssertHidden:
		return true
	default:
		return false
	}
}

func isValidScrollPosition(p ScrollPosition) bool {
	switch p {
	case ScrollTop, ScrollBottom:
		return true
	default:
		return false
	}
}
