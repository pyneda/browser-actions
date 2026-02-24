package browseractions

import "testing"

func TestValidateStrictWaitLoadWithoutSelector(t *testing.T) {
	err := ValidateScript(BrowserActions{
		Title: "wait load",
		Actions: []Action{
			{Type: ActionWait, For: WaitLoad},
		},
	}, ValidationProfileStrict)

	if err != nil {
		t.Fatalf("expected strict validation to allow wait(load) without selector, got %v", err)
	}
}

func TestValidateStrictWaitVisibleRequiresSelector(t *testing.T) {
	err := ValidateScript(BrowserActions{
		Title: "wait visible",
		Actions: []Action{
			{Type: ActionWait, For: WaitVisible},
		},
	}, ValidationProfileStrict)

	if err == nil {
		t.Fatalf("expected strict validation error for wait(visible) without selector")
	}
}

func TestValidateLegacyScreenshotRequiresFile(t *testing.T) {
	err := ValidateActions([]Action{{Type: ActionScreenshot}}, ValidationProfileLenient)
	if err == nil {
		t.Fatalf("expected legacy validation to require screenshot file")
	}
	de := requireDiagnosticsError(t, err)
	requireDiagnostic(t, de.Diagnostics, "actions.0.file", "required")
}

func TestValidateStrictScreenshotDoesNotRequireFile(t *testing.T) {
	err := ValidateActions([]Action{{Type: ActionScreenshot}}, ValidationProfileStrict)
	if err != nil {
		t.Fatalf("expected strict validation to allow screenshot without file, got %v", err)
	}
}

func TestValidateStrictAssertContainsRequiresValue(t *testing.T) {
	err := ValidateActions([]Action{{
		Type:      ActionAssert,
		Selector:  "#x",
		Condition: AssertContains,
	}}, ValidationProfileStrict)
	if err == nil {
		t.Fatalf("expected strict validation to require value for contains assertion")
	}
}

func TestValidateStrictAssertVisibleDoesNotRequireValue(t *testing.T) {
	err := ValidateActions([]Action{{
		Type:      ActionAssert,
		Selector:  "#x",
		Condition: AssertVisible,
	}}, ValidationProfileStrict)
	if err != nil {
		t.Fatalf("expected strict validation to allow visible assertion without value, got %v", err)
	}
}

func TestValidateStrictAssertEqualsRequiresValueAndHiddenDoesNot(t *testing.T) {
	tests := []struct {
		name      string
		condition AssertCondition
		wantErr   bool
	}{
		{name: "equals requires value", condition: AssertEquals, wantErr: true},
		{name: "hidden does not require value", condition: AssertHidden, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateActions([]Action{{
				Type:      ActionAssert,
				Selector:  "#x",
				Condition: tt.condition,
			}}, ValidationProfileStrict)

			if tt.wantErr && err == nil {
				t.Fatalf("expected error")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestValidateWaitRulesDifferBetweenStrictAndLegacy(t *testing.T) {
	action := Action{Type: ActionWait, For: WaitVisible}

	if err := ValidateActions([]Action{action}, ValidationProfileLenient); err != nil {
		t.Fatalf("expected legacy profile to allow wait visible without selector, got %v", err)
	}

	err := ValidateActions([]Action{action}, ValidationProfileStrict)
	de := requireDiagnosticsError(t, err)
	requireDiagnostic(t, de.Diagnostics, "actions.0.selector", "required")
}

func TestValidateStrictWaitRequiresCondition(t *testing.T) {
	err := ValidateActions([]Action{{Type: ActionWait, Selector: "#x"}}, ValidationProfileStrict)
	de := requireDiagnosticsError(t, err)
	requireDiagnostic(t, de.Diagnostics, "actions.0.for", "required")
}

func TestValidateUnknownProfileReturnsError(t *testing.T) {
	err := ValidateActions([]Action{{Type: ActionSleep, Duration: 1}}, ValidationProfile("unknown"))
	if err == nil {
		t.Fatalf("expected error for unknown profile")
	}
	de := requireDiagnosticsError(t, err)
	requireDiagnostic(t, de.Diagnostics, "profile", "invalid")
}

func TestValidateInvalidEnumsProduceClearDiagnostics(t *testing.T) {
	err := ValidateActions([]Action{
		{Type: ActionWait, Selector: "#x", For: WaitCondition("later")},
		{Type: ActionAssert, Selector: "#x", Condition: AssertCondition("maybe"), Value: "v"},
		{Type: ActionScroll, Position: ScrollPosition("middle")},
	}, ValidationProfileStrict)

	de := requireDiagnosticsError(t, err)
	requireDiagnosticMessageContains(t, de.Diagnostics, "actions.0.for", "oneof", "visible/hidden/enabled/load")
	requireDiagnosticMessageContains(t, de.Diagnostics, "actions.1.condition", "oneof", "contains/equals/visible/hidden")
	requireDiagnosticMessageContains(t, de.Diagnostics, "actions.2.position", "oneof", "top/bottom")
}

func TestValidateScriptAggregatesTopLevelAndActionDiagnostics(t *testing.T) {
	err := ValidateScript(BrowserActions{
		Title: "",
		Actions: []Action{
			{Type: ActionFill, Selector: "", Value: ""},
		},
	}, ValidationProfileStrict)
	de := requireDiagnosticsError(t, err)
	requireDiagnostic(t, de.Diagnostics, "title", "required")
	requireDiagnostic(t, de.Diagnostics, "actions.0.selector", "required")
	requireDiagnostic(t, de.Diagnostics, "actions.0.value", "required")
}
