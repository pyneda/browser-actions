package cli

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	browseractions "github.com/pyneda/browser-actions"
	"github.com/pyneda/browser-actions/rodexec"
)

func TestExecuteRootDispatchUsageAndUnknownCommand(t *testing.T) {
	code, stdout, stderr := runCLI(t, "", []string{}...)
	if code != 2 {
		t.Fatalf("expected exit 2 for no args, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected no stdout for no args, got %q", stdout)
	}
	if !strings.Contains(stderr, "browser-actions <command>") {
		t.Fatalf("expected root usage on stderr, got %q", stderr)
	}

	code, stdout, stderr = runCLI(t, "", "help")
	if code != 0 {
		t.Fatalf("expected exit 0 for help, got %d", code)
	}
	if !strings.Contains(stdout, "Commands:") {
		t.Fatalf("expected help output, got %q", stdout)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr for help, got %q", stderr)
	}

	code, _, stderr = runCLI(t, "", "wat")
	if code != 2 {
		t.Fatalf("expected exit 2 for unknown command, got %d", code)
	}
	if !strings.Contains(stderr, `unknown command "wat"`) {
		t.Fatalf("expected unknown command error, got %q", stderr)
	}
}

func TestValidateCommandHumanAndJSONOutput(t *testing.T) {
	dir := t.TempDir()
	validPath := writeTempFile(t, dir, "valid.yaml", `
title: CLI Validate
actions:
  - type: sleep
    duration: 1
`)

	code, stdout, stderr := runCLI(t, "", "validate", validPath)
	if code != 0 {
		t.Fatalf("expected validate success exit code, got %d stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "Browser Actions Validation") ||
		!strings.Contains(stdout, "Status") ||
		!strings.Contains(stdout, "VALID") {
		t.Fatalf("unexpected human validate output:\n%s", stdout)
	}

	code, stdout, stderr = runCLI(t, "", "validate", "--json", validPath)
	if code != 0 {
		t.Fatalf("expected validate json success exit code, got %d stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr for json output, got %q", stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("failed to parse validate json output %q: %v", stdout, err)
	}
	if valid, _ := out["valid"].(bool); !valid {
		t.Fatalf("expected valid=true, got %+v", out)
	}
	if out["title"] != "CLI Validate" {
		t.Fatalf("unexpected title in output: %+v", out)
	}
	if actions, ok := out["actions"].(float64); !ok || int(actions) != 1 {
		t.Fatalf("unexpected actions count: %+v", out)
	}
	if _, ok := out["diagnostics"]; !ok {
		t.Fatalf("expected diagnostics field in validate json output")
	}
}

func TestValidateCommandFailuresAndStdin(t *testing.T) {
	dir := t.TempDir()
	parseFailPath := writeTempFile(t, dir, "broken.json", `{"title":}`)
	invalidPath := writeTempFile(t, dir, "invalid.yaml", `
title: Invalid
actions:
  - type: wait
    for: visible
`)

	code, stdout, stderr := runCLI(t, "", "validate", parseFailPath)
	if code != 1 {
		t.Fatalf("expected parse failure exit code 1, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected no stdout for human parse failure, got %q", stdout)
	}
	if !strings.Contains(stderr, "Browser Actions Validation") ||
		!strings.Contains(stderr, "Stage") ||
		!strings.Contains(stderr, "parse") ||
		!strings.Contains(stderr, "Diagnostics") {
		t.Fatalf("unexpected human parse failure output:\n%s", stderr)
	}

	code, stdout, stderr = runCLI(t, "", "validate", "--json", invalidPath)
	if code != 1 {
		t.Fatalf("expected validation failure exit code 1, got %d", code)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr for json validation failure, got %q", stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("failed to parse validate failure json %q: %v", stdout, err)
	}
	if valid, _ := out["valid"].(bool); valid {
		t.Fatalf("expected valid=false, got %+v", out)
	}
	if _, ok := out["error"].(string); !ok {
		t.Fatalf("expected error string in validate failure output: %+v", out)
	}
	if _, ok := out["diagnostics"].([]any); !ok {
		t.Fatalf("expected diagnostics array in validate failure output: %+v", out)
	}

	stdinInvalid := `
title: Stdin Validate
actions:
  - type: evaluate
`
	code, stdout, stderr = runCLI(t, stdinInvalid, "validate", "--json", "-")
	if code != 1 {
		t.Fatalf("expected stdin validate failure exit code 1, got %d", code)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr for stdin json validate, got %q", stderr)
	}
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("failed to parse stdin validate json %q: %v", stdout, err)
	}
	if valid, _ := out["valid"].(bool); valid {
		t.Fatalf("expected stdin validate valid=false, got %+v", out)
	}
}

func TestValidateCommandInvalidArgsAndFlags(t *testing.T) {
	code, _, stderr := runCLI(t, "", "validate")
	if code != 2 {
		t.Fatalf("expected exit 2 for missing validate arg, got %d", code)
	}
	if !strings.Contains(stderr, "usage: browser-actions validate") {
		t.Fatalf("expected validate usage on stderr, got %q", stderr)
	}

	code, _, stderr = runCLI(t, "", "validate", "--does-not-exist")
	if code != 2 {
		t.Fatalf("expected exit 2 for bad validate flag, got %d", code)
	}
	if !strings.Contains(stderr, "flag provided but not defined") {
		t.Fatalf("expected flag parse error, got %q", stderr)
	}
}

func TestRunCommandValidationFailureHumanAndJSON(t *testing.T) {
	dir := t.TempDir()
	invalidPath := writeTempFile(t, dir, "invalid.yaml", `
title: Invalid Run
actions:
  - type: wait
    for: visible
`)

	code, stdout, stderr := runCLI(t, "", "run", invalidPath)
	if code != 1 {
		t.Fatalf("expected run validation failure exit code 1, got %d stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected human run failure on stdout, got stderr=%q", stderr)
	}
	for _, want := range []string{"Browser Actions Run", "Script", "Result", "Diagnostics", "Run error:"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %q in human run failure output:\n%s", want, stdout)
		}
	}

	code, stdout, stderr = runCLI(t, "", "run", "--json", invalidPath)
	if code != 1 {
		t.Fatalf("expected run json validation failure exit code 1, got %d", code)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr for run json output, got %q", stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("failed to parse run failure json %q: %v", stdout, err)
	}
	if okVal, _ := out["ok"].(bool); okVal {
		t.Fatalf("expected ok=false, got %+v", out)
	}
	if _, ok := out["error"].(string); !ok {
		t.Fatalf("expected error string in run failure json: %+v", out)
	}
	if _, ok := out["diagnostics"].([]any); !ok {
		t.Fatalf("expected diagnostics array in run failure json: %+v", out)
	}
	if _, ok := out["result"].(map[string]any); !ok {
		t.Fatalf("expected result object in run failure json: %+v", out)
	}
}

func TestRunCommandStdinValidationFailurePath(t *testing.T) {
	stdinInvalid := `
title: Stdin Run
actions:
  - type: wait
    for: visible
`
	code, stdout, stderr := runCLI(t, stdinInvalid, "run", "--json", "-")
	if code != 1 {
		t.Fatalf("expected run stdin validation failure exit code 1, got %d stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("failed to parse run stdin json %q: %v", stdout, err)
	}
	if okVal, _ := out["ok"].(bool); okVal {
		t.Fatalf("expected ok=false, got %+v", out)
	}
}

func TestRunCommandSuccessHumanAndJSONWithStubbedRunner(t *testing.T) {
	dir := t.TempDir()
	scriptPath := writeTempFile(t, dir, "valid.yaml", `
title: Stubbed Run
actions:
  - type: sleep
    duration: 1
`)

	restore := stubRunScript(t, func(ctx context.Context, script browseractions.BrowserActions, opts *rodexec.RunOptions) (rodexec.ExecutionResult, error) {
		if script.Title != "Stubbed Run" {
			t.Fatalf("unexpected script in stub: %+v", script)
		}
		return rodexec.ExecutionResult{
			Succeeded:        true,
			TotalActions:     len(script.Actions),
			CompletedActions: len(script.Actions),
			Screenshots:      []rodexec.ScreenshotResult{{OutputFile: filepath.Join(dir, "shot.png")}},
			Evaluations:      []rodexec.EvaluationResult{{Expression: "() => 1", Value: "1"}},
			Logs:             []rodexec.LogEntry{{Level: rodexec.LogLevelInfo, Text: "stubbed", Timestamp: time.Unix(0, 0)}},
			Duration:         7 * time.Millisecond,
		}, nil
	})
	defer restore()

	code, stdout, stderr := runCLI(t, "", "run", "--json", scriptPath)
	if code != 0 {
		t.Fatalf("expected run json success exit code 0, got %d stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr for run json success, got %q", stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("failed to parse run success json %q: %v", stdout, err)
	}
	if okVal, _ := out["ok"].(bool); !okVal {
		t.Fatalf("expected ok=true, got %+v", out)
	}
	scriptObj, ok := out["script"].(map[string]any)
	if !ok || scriptObj["title"] != "Stubbed Run" {
		t.Fatalf("unexpected script object: %+v", out["script"])
	}
	resultObj, ok := out["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected result object, got %+v", out["result"])
	}
	if succeeded, _ := resultObj["succeeded"].(bool); !succeeded {
		t.Fatalf("expected result.succeeded=true, got %+v", resultObj)
	}

	code, stdout, stderr = runCLI(t, "", "run", scriptPath)
	if code != 0 {
		t.Fatalf("expected run human success exit code 0, got %d stderr=%q", code, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr for run human success, got %q", stderr)
	}
	for _, want := range []string{"Browser Actions Run", "Script", "Result", "Artifacts", "Evaluations", "Recent Logs", "Status", "OK"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %q in run human output:\n%s", want, stdout)
		}
	}
}

func TestRunCommandPropagatesFlagsToRunner(t *testing.T) {
	dir := t.TempDir()
	scriptPath := writeTempFile(t, dir, "valid.yaml", `
title: Flag Run
actions:
  - type: sleep
    duration: 1
`)

	var (
		gotScript browseractions.BrowserActions
		gotOpts   *rodexec.RunOptions
	)
	restore := stubRunScript(t, func(ctx context.Context, script browseractions.BrowserActions, opts *rodexec.RunOptions) (rodexec.ExecutionResult, error) {
		gotScript = script
		copyOpts := *opts
		gotOpts = &copyOpts
		return rodexec.ExecutionResult{Succeeded: true, TotalActions: 1, CompletedActions: 1}, nil
	})
	defer restore()

	code, stdout, stderr := runCLI(t, "", "run",
		"--json",
		"--timeout", "123ms",
		"--headed",
		"--viewport", "800x600",
		"--proxy", "http://127.0.0.1:8080",
		"--binary", "/tmp/fake-browser",
		"--profile", "lenient",
		"--screenshot-data", "inline",
		"--output-dir", "artifacts",
		"--format", "yaml",
		scriptPath,
	)
	if code != 0 {
		t.Fatalf("expected run success exit code 0, got %d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if gotScript.Title != "Flag Run" {
		t.Fatalf("unexpected script passed to runner: %+v", gotScript)
	}
	if gotOpts == nil {
		t.Fatalf("expected runner options")
	}
	if gotOpts.Headless {
		t.Fatalf("expected headed mode to set Headless=false")
	}
	if gotOpts.Timeout != 123*time.Millisecond {
		t.Fatalf("unexpected timeout: %v", gotOpts.Timeout)
	}
	if gotOpts.Proxy != "http://127.0.0.1:8080" {
		t.Fatalf("unexpected proxy: %q", gotOpts.Proxy)
	}
	if gotOpts.BinaryPath != "/tmp/fake-browser" {
		t.Fatalf("unexpected binary path: %q", gotOpts.BinaryPath)
	}
	if gotOpts.Viewport == nil || gotOpts.Viewport.Width != 800 || gotOpts.Viewport.Height != 600 {
		t.Fatalf("unexpected viewport: %+v", gotOpts.Viewport)
	}
	if gotOpts.ExecuteOptions == nil {
		t.Fatalf("expected ExecuteOptions to be set")
	}
	if gotOpts.ExecuteOptions.ValidationProfile != browseractions.ValidationProfileLenient {
		t.Fatalf("unexpected profile: %q", gotOpts.ExecuteOptions.ValidationProfile)
	}
	if !gotOpts.ExecuteOptions.IncludeScreenshotData {
		t.Fatalf("expected inline screenshot mode")
	}
	if !gotOpts.ExecuteOptions.WriteFiles {
		t.Fatalf("expected WriteFiles=true in CLI run options")
	}
	if gotOpts.ExecuteOptions.OutputDir != "artifacts" {
		t.Fatalf("unexpected output dir: %q", gotOpts.ExecuteOptions.OutputDir)
	}
	if !gotOpts.ExecuteOptions.Validate {
		t.Fatalf("expected CLI to request validation")
	}
}

func TestRunCommandInvalidArgsAndFlags(t *testing.T) {
	dir := t.TempDir()
	scriptPath := writeTempFile(t, dir, "valid.yaml", `
title: x
actions:
  - type: sleep
    duration: 1
`)

	code, stdout, stderr := runCLI(t, "", "run", "--viewport", "bad", scriptPath)
	if code != 2 {
		t.Fatalf("expected exit 2 for invalid viewport, got %d", code)
	}
	if stdout != "" {
		t.Fatalf("expected no stdout for human cli error, got %q", stdout)
	}
	if !strings.Contains(stderr, "expected WIDTHxHEIGHT") {
		t.Fatalf("expected viewport error message, got %q", stderr)
	}

	code, stdout, stderr = runCLI(t, "", "run", "--json", "--screenshot-data", "file", scriptPath)
	if code != 2 {
		t.Fatalf("expected exit 2 for invalid screenshot-data, got %d", code)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr for json cli error, got %q", stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("failed to parse cli json error %q: %v", stdout, err)
	}
	if okVal, _ := out["ok"].(bool); okVal {
		t.Fatalf("expected ok=false for invalid screenshot-data, got %+v", out)
	}
	if !strings.Contains(stringValue(out["error"]), "invalid --screenshot-data") {
		t.Fatalf("unexpected screenshot-data error output: %+v", out)
	}

	code, _, stderr = runCLI(t, "", "run")
	if code != 2 {
		t.Fatalf("expected exit 2 for missing run arg, got %d", code)
	}
	if !strings.Contains(stderr, "usage: browser-actions run") {
		t.Fatalf("expected run usage on stderr, got %q", stderr)
	}
}

func TestCapabilitiesCommandHumanAndJSON(t *testing.T) {
	code, stdout, stderr := runCLI(t, "", "capabilities")
	if code != 0 {
		t.Fatalf("expected capabilities success exit code, got %d", code)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}
	for _, want := range []string{"Browser Actions Capabilities", "Supported Actions", "Validation Profiles"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected %q in capabilities output:\n%s", want, stdout)
		}
	}

	code, stdout, stderr = runCLI(t, "", "capabilities", "--json")
	if code != 0 {
		t.Fatalf("expected capabilities json success exit code, got %d", code)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr for capabilities json, got %q", stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("failed to parse capabilities json %q: %v", stdout, err)
	}
	if out["backend"] != "rod" {
		t.Fatalf("unexpected backend: %+v", out)
	}
	if _, ok := out["actions"].([]any); !ok {
		t.Fatalf("expected actions array in capabilities json: %+v", out)
	}
	if _, ok := out["profiles"].([]any); !ok {
		t.Fatalf("expected profiles array in capabilities json: %+v", out)
	}
}

func TestVersionCommandHumanAndJSON(t *testing.T) {
	origVersion := Version
	Version = "test-version"
	defer func() { Version = origVersion }()

	code, stdout, stderr := runCLI(t, "", "version")
	if code != 0 {
		t.Fatalf("expected version success exit code, got %d", code)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "browser-actions") || !strings.Contains(stdout, "test-version") {
		t.Fatalf("unexpected version human output:\n%s", stdout)
	}

	code, stdout, stderr = runCLI(t, "", "version", "--json")
	if code != 0 {
		t.Fatalf("expected version json success exit code, got %d", code)
	}
	if stderr != "" {
		t.Fatalf("expected no stderr for version json, got %q", stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("failed to parse version json %q: %v", stdout, err)
	}
	if out["version"] != "test-version" {
		t.Fatalf("unexpected version json output: %+v", out)
	}
}

func runCLI(t *testing.T, stdin string, args ...string) (int, string, string) {
	t.Helper()

	oldStdout, oldStderr, oldStdin := os.Stdout, os.Stderr, os.Stdin
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stdout: %v", err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe stderr: %v", err)
	}

	inFile, err := os.CreateTemp(t.TempDir(), "stdin-*")
	if err != nil {
		t.Fatalf("CreateTemp stdin: %v", err)
	}
	if _, err := inFile.WriteString(stdin); err != nil {
		t.Fatalf("write stdin temp file: %v", err)
	}
	if _, err := inFile.Seek(0, 0); err != nil {
		t.Fatalf("seek stdin temp file: %v", err)
	}

	os.Stdout = outW
	os.Stderr = errW
	os.Stdin = inFile

	code := Execute(context.Background(), args)

	_ = outW.Close()
	_ = errW.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	os.Stdin = oldStdin

	stdoutBytes, err := io.ReadAll(outR)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	stderrBytes, err := io.ReadAll(errR)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	_ = outR.Close()
	_ = errR.Close()
	_ = inFile.Close()

	return code, string(stdoutBytes), string(stderrBytes)
}

func writeTempFile(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(strings.TrimLeft(contents, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(%s): %v", path, err)
	}
	return path
}

func stubRunScript(t *testing.T, fn func(context.Context, browseractions.BrowserActions, *rodexec.RunOptions) (rodexec.ExecutionResult, error)) func() {
	t.Helper()
	orig := runScript
	runScript = fn
	return func() { runScript = orig }
}

func stringValue(v any) string {
	s, _ := v.(string)
	return s
}
