package browseractions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadFile(path string, opts ParseOptions) (BrowserActions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BrowserActions{}, err
	}
	return Parse(data, opts)
}

func LoadReader(r io.Reader, opts ParseOptions) (BrowserActions, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return BrowserActions{}, err
	}
	return Parse(data, opts)
}

func Parse(data []byte, opts ParseOptions) (BrowserActions, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return BrowserActions{}, diagnosticsErr("parse", nil, Diagnostic{
			Kind:    "parse",
			Code:    "empty_input",
			Message: "input is empty",
		})
	}

	format := opts.Format
	if format == "" || format == InputFormatAuto {
		format = detectFormat(trimmed)
	}

	switch format {
	case InputFormatJSON:
		return parseJSON(trimmed, opts)
	case InputFormatYAML:
		return parseYAML(trimmed, opts)
	default:
		return BrowserActions{}, diagnosticsErr("parse", nil, Diagnostic{
			Kind:    "parse",
			Code:    "invalid_format",
			Path:    "format",
			Message: fmt.Sprintf("unsupported format %q", format),
		})
	}
}

func detectFormat(trimmed []byte) InputFormat {
	if len(trimmed) == 0 {
		return InputFormatAuto
	}
	switch trimmed[0] {
	case '{', '[':
		return InputFormatJSON
	default:
		return InputFormatYAML
	}
}

func parseJSON(data []byte, opts ParseOptions) (BrowserActions, error) {
	if len(data) > 0 && data[0] == '[' {
		var actions []Action
		if err := json.Unmarshal(data, &actions); err != nil {
			return BrowserActions{}, diagnosticsErr("parse", err, Diagnostic{
				Kind:    "parse",
				Code:    "invalid_json",
				Message: "failed to parse JSON actions array",
			})
		}
		return wrapActionsArray(actions, opts), nil
	}

	var script BrowserActions
	if err := json.Unmarshal(data, &script); err != nil {
		return BrowserActions{}, diagnosticsErr("parse", err, Diagnostic{
			Kind:    "parse",
			Code:    "invalid_json",
			Message: "failed to parse JSON browser actions object",
		})
	}
	return script, nil
}

func parseYAML(data []byte, opts ParseOptions) (BrowserActions, error) {
	// Simple top-level sequence detection for common YAML action-array inputs.
	if len(data) > 0 && data[0] == '-' {
		var actions []Action
		if err := yaml.Unmarshal(data, &actions); err == nil {
			return wrapActionsArray(actions, opts), nil
		}
	}

	var script BrowserActions
	if err := yaml.Unmarshal(data, &script); err != nil {
		return BrowserActions{}, diagnosticsErr("parse", err, Diagnostic{
			Kind:    "parse",
			Code:    "invalid_yaml",
			Message: "failed to parse YAML browser actions object",
		})
	}

	if script.Title == "" && len(script.Actions) == 0 && looksLikeYAMLSequence(data) {
		var actions []Action
		if err := yaml.Unmarshal(data, &actions); err == nil {
			return wrapActionsArray(actions, opts), nil
		}
	}

	return script, nil
}

func looksLikeYAMLSequence(data []byte) bool {
	s := strings.TrimSpace(string(data))
	return strings.HasPrefix(s, "- ")
}

func wrapActionsArray(actions []Action, opts ParseOptions) BrowserActions {
	title := strings.TrimSpace(opts.ArrayTitle)
	if title == "" {
		title = "Untitled"
	}
	return BrowserActions{
		Title:   title,
		Actions: actions,
	}
}
