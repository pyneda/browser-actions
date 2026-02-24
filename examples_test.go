package browseractions

import (
	"path/filepath"
	"testing"
)

func TestExamplesValidateInStrictProfile(t *testing.T) {
	t.Parallel()

	files := []string{
		"examples/basic-navigation.yaml",
		"examples/dom-playground.yaml",
		"examples/page-scroll.json",
		"examples/raw-array.json",
		"examples/offline-local.yaml",
	}

	for _, path := range files {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()

			script, err := LoadFile(path, ParseOptions{})
			if err != nil {
				t.Fatalf("LoadFile(%s) error: %v", path, err)
			}
			if err := ValidateScript(script, ValidationProfileStrict); err != nil {
				t.Fatalf("ValidateScript(%s) error: %v", path, err)
			}
		})
	}
}

func TestInvalidExampleFailsStrictValidation(t *testing.T) {
	script, err := LoadFile("examples/invalid-demo.yaml", ParseOptions{})
	if err != nil {
		t.Fatalf("LoadFile returned error: %v", err)
	}

	err = ValidateScript(script, ValidationProfileStrict)
	de := requireDiagnosticsError(t, err)
	requireDiagnostic(t, de.Diagnostics, "actions.0.selector", "required")
	requireDiagnostic(t, de.Diagnostics, "actions.2.expression", "required")
}
