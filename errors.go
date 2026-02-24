package browseractions

import (
	"errors"
	"fmt"
	"strings"
)

type Diagnostic struct {
	Kind    string `json:"kind"`
	Path    string `json:"path,omitempty"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type DiagnosticsError struct {
	Kind        string       `json:"kind"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Cause       error        `json:"-"`
}

func (e *DiagnosticsError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if len(e.Diagnostics) == 0 {
		if e.Cause != nil {
			return fmt.Sprintf("%s error: %v", e.Kind, e.Cause)
		}
		return e.Kind + " error"
	}
	msgs := make([]string, 0, len(e.Diagnostics))
	for _, d := range e.Diagnostics {
		if d.Path != "" {
			msgs = append(msgs, d.Path+": "+d.Message)
			continue
		}
		msgs = append(msgs, d.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Kind, strings.Join(msgs, "; "))
}

func (e *DiagnosticsError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func AsDiagnosticsError(err error) (*DiagnosticsError, bool) {
	var de *DiagnosticsError
	if errors.As(err, &de) {
		return de, true
	}
	return nil, false
}

func diagnosticsErr(kind string, cause error, diagnostics ...Diagnostic) error {
	return &DiagnosticsError{
		Kind:        kind,
		Diagnostics: diagnostics,
		Cause:       cause,
	}
}
