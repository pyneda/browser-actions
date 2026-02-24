package browseractions

import "testing"

func TestParseJSONObject(t *testing.T) {
	data := []byte(`{"title":"JSON object","actions":[{"type":"sleep","duration":1}]}`)

	script, err := Parse(data, ParseOptions{})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if script.Title != "JSON object" {
		t.Fatalf("unexpected title: %q", script.Title)
	}
	if len(script.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(script.Actions))
	}
	if script.Actions[0].Type != ActionSleep {
		t.Fatalf("unexpected action type: %q", script.Actions[0].Type)
	}
}

func TestParseJSONArrayWrapsActions(t *testing.T) {
	data := []byte(`[{"type":"click","selector":"#btn"}]`)

	script, err := Parse(data, ParseOptions{})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if script.Title != "Untitled" {
		t.Fatalf("expected title Untitled, got %q", script.Title)
	}
	if len(script.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(script.Actions))
	}
	if script.Actions[0].Type != ActionClick {
		t.Fatalf("expected click action, got %q", script.Actions[0].Type)
	}
}

func TestParseYAMLObject(t *testing.T) {
	data := []byte(`
title: Login flow
actions:
  - type: navigate
    url: https://example.com
`)

	script, err := Parse(data, ParseOptions{})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if script.Title != "Login flow" {
		t.Fatalf("unexpected title: %q", script.Title)
	}
	if len(script.Actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(script.Actions))
	}
	if script.Actions[0].Type != ActionNavigate {
		t.Fatalf("expected navigate action, got %q", script.Actions[0].Type)
	}
}

func TestParseYAMLArrayWrapsActions(t *testing.T) {
	data := []byte(`
- type: wait
  for: load
- type: sleep
  duration: 10
`)

	script, err := Parse(data, ParseOptions{ArrayTitle: "Array Example"})
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if script.Title != "Array Example" {
		t.Fatalf("unexpected title: %q", script.Title)
	}
	if len(script.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(script.Actions))
	}
}

func TestParseEmptyInputReturnsDiagnostics(t *testing.T) {
	_, err := Parse([]byte("   "), ParseOptions{})
	if err == nil {
		t.Fatalf("expected error for empty input")
	}
	de, ok := AsDiagnosticsError(err)
	if !ok {
		t.Fatalf("expected DiagnosticsError, got %T", err)
	}
	if len(de.Diagnostics) == 0 || de.Diagnostics[0].Code != "empty_input" {
		t.Fatalf("expected empty_input diagnostic, got %+v", de.Diagnostics)
	}
}

func TestParseInvalidFormatOption(t *testing.T) {
	_, err := Parse([]byte(`{"title":"x","actions":[]}`), ParseOptions{Format: InputFormat("toml")})
	if err == nil {
		t.Fatalf("expected invalid format error")
	}
	de := requireDiagnosticsError(t, err)
	requireDiagnostic(t, de.Diagnostics, "format", "invalid_format")
}

func TestParseInvalidJSONReturnsDiagnostics(t *testing.T) {
	_, err := Parse([]byte(`{"title":}`), ParseOptions{Format: InputFormatJSON})
	de := requireDiagnosticsError(t, err)
	if de.Kind != "parse" {
		t.Fatalf("expected parse diagnostics kind, got %q", de.Kind)
	}
	if len(de.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(de.Diagnostics))
	}
	d := de.Diagnostics[0]
	if d.Kind != "parse" || d.Code != "invalid_json" {
		t.Fatalf("unexpected diagnostic: %+v", d)
	}
	if d.Message == "" {
		t.Fatalf("expected diagnostic message")
	}
	if de.Cause == nil {
		t.Fatalf("expected wrapped parse cause")
	}
}

func TestParseInvalidYAMLReturnsDiagnostics(t *testing.T) {
	_, err := Parse([]byte("title: [broken\n"), ParseOptions{Format: InputFormatYAML})
	de := requireDiagnosticsError(t, err)
	if de.Kind != "parse" {
		t.Fatalf("expected parse diagnostics kind, got %q", de.Kind)
	}
	if len(de.Diagnostics) != 1 {
		t.Fatalf("expected 1 diagnostic, got %d", len(de.Diagnostics))
	}
	d := de.Diagnostics[0]
	if d.Kind != "parse" || d.Code != "invalid_yaml" {
		t.Fatalf("unexpected diagnostic: %+v", d)
	}
	if d.Message == "" {
		t.Fatalf("expected diagnostic message")
	}
	if de.Cause == nil {
		t.Fatalf("expected wrapped parse cause")
	}
}

func TestParseFormatOverrideForcesJSON(t *testing.T) {
	data := []byte("title: YAML document\nactions:\n  - type: sleep\n    duration: 1\n")

	_, err := Parse(data, ParseOptions{Format: InputFormatJSON})
	de := requireDiagnosticsError(t, err)
	if de.Diagnostics[0].Code != "invalid_json" {
		t.Fatalf("expected invalid_json diagnostic, got %+v", de.Diagnostics)
	}
}

func TestParseFormatOverrideYAMLCanParseJSONObject(t *testing.T) {
	data := []byte(`{"title":"json-via-yaml","actions":[{"type":"sleep","duration":2}]}`)

	script, err := Parse(data, ParseOptions{Format: InputFormatYAML})
	if err != nil {
		t.Fatalf("expected YAML override to parse JSON object (YAML superset), got %v", err)
	}
	if script.Title != "json-via-yaml" {
		t.Fatalf("unexpected title: %q", script.Title)
	}
	if len(script.Actions) != 1 || script.Actions[0].Type != ActionSleep {
		t.Fatalf("unexpected actions: %+v", script.Actions)
	}
}
