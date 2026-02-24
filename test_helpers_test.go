package browseractions

import (
	"strings"
	"testing"
)

func requireDiagnosticsError(t *testing.T, err error) *DiagnosticsError {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error")
	}
	de, ok := AsDiagnosticsError(err)
	if !ok {
		t.Fatalf("expected DiagnosticsError, got %T (%v)", err, err)
	}
	return de
}

func requireDiagnostic(t *testing.T, diags []Diagnostic, path, code string) Diagnostic {
	t.Helper()
	for _, d := range diags {
		if d.Path == path && d.Code == code {
			return d
		}
	}
	t.Fatalf("expected diagnostic path=%q code=%q, got %+v", path, code, diags)
	return Diagnostic{}
}

func requireDiagnosticMessageContains(t *testing.T, diags []Diagnostic, path, code, wantSubstr string) {
	t.Helper()
	d := requireDiagnostic(t, diags, path, code)
	if !strings.Contains(d.Message, wantSubstr) {
		t.Fatalf("expected diagnostic message for %s/%s to contain %q, got %q", path, code, wantSubstr, d.Message)
	}
}
