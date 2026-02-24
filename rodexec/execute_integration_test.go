package rodexec

import (
	"context"
	"encoding/base64"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	browseractions "github.com/pyneda/browser-actions"
)

func TestExecuteOfflineSuccessFlow(t *testing.T) {
	env := newRodTestEnv(t, fixtureFlowHTML)
	defer env.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	actions := []browseractions.Action{
		{Type: browseractions.ActionNavigate, URL: env.URL()},
		{Type: browseractions.ActionWait, For: browseractions.WaitLoad}, // regression: no selector allowed
		{Type: browseractions.ActionAssert, Selector: "#form", Condition: browseractions.AssertHidden},
		{Type: browseractions.ActionClick, Selector: "#show"},
		{Type: browseractions.ActionWait, Selector: "#form", For: browseractions.WaitVisible},
		{Type: browseractions.ActionFill, Selector: "#name", Value: "Alice"},
		{Type: browseractions.ActionClick, Selector: "#submit"},
		{Type: browseractions.ActionAssert, Selector: "#status", Condition: browseractions.AssertVisible},
		{Type: browseractions.ActionAssert, Selector: "#status", Condition: browseractions.AssertContains, Value: "Alice"},
		{Type: browseractions.ActionEvaluate, Expression: `() => document.querySelector('#status').textContent.includes('Alice')`},
	}

	result, err := Execute(ctx, env.page, actions, &ExecuteOptions{
		Validate:          true,
		ValidationProfile: browseractions.ValidationProfileStrict,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !result.Succeeded {
		t.Fatalf("expected success result, got %+v", result)
	}
	if result.CompletedActions != len(actions) {
		t.Fatalf("expected %d completed actions, got %d", len(actions), result.CompletedActions)
	}
	if result.Failure != nil {
		t.Fatalf("unexpected failure: %+v", result.Failure)
	}
	if len(result.Evaluations) != 1 {
		t.Fatalf("expected 1 evaluation, got %d", len(result.Evaluations))
	}
	if strings.TrimSpace(result.Evaluations[0].Value) != "true" {
		t.Fatalf("expected evaluation result true, got %q", result.Evaluations[0].Value)
	}
}

func TestExecuteWaitLoadWithoutSelector(t *testing.T) {
	env := newRodTestEnv(t, fixtureFlowHTML)
	defer env.Close()

	actions := []browseractions.Action{
		{Type: browseractions.ActionNavigate, URL: env.URL()},
		{Type: browseractions.ActionWait, For: browseractions.WaitLoad},
	}

	result, err := Execute(context.Background(), env.page, actions, &ExecuteOptions{
		Validate:          true,
		ValidationProfile: browseractions.ValidationProfileStrict,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.CompletedActions != 2 || !result.Succeeded {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestExecuteScrollPageTopAndBottomWithoutSelector(t *testing.T) {
	env := newRodTestEnv(t, fixtureScrollHTML)
	defer env.Close()

	actions := []browseractions.Action{
		{Type: browseractions.ActionNavigate, URL: env.URL()},
		{Type: browseractions.ActionWait, For: browseractions.WaitLoad},
		{Type: browseractions.ActionScroll, Position: browseractions.ScrollBottom},
		{Type: browseractions.ActionEvaluate, Expression: `() => window.scrollY`},
		{Type: browseractions.ActionScroll, Position: browseractions.ScrollTop},
		{Type: browseractions.ActionEvaluate, Expression: `() => window.scrollY`},
	}

	result, err := Execute(context.Background(), env.page, actions, &ExecuteOptions{
		Validate:          true,
		ValidationProfile: browseractions.ValidationProfileStrict,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(result.Evaluations) != 2 {
		t.Fatalf("expected 2 evaluations, got %d", len(result.Evaluations))
	}

	bottomY := mustAtoi(t, result.Evaluations[0].Value)
	topY := mustAtoi(t, result.Evaluations[1].Value)
	if bottomY <= 0 {
		t.Fatalf("expected scrollY > 0 after bottom scroll, got %d", bottomY)
	}
	if topY != 0 {
		t.Fatalf("expected scrollY = 0 after top scroll, got %d", topY)
	}
}

func TestExecuteScrollElementPositionTopAndBottom(t *testing.T) {
	env := newRodTestEnv(t, fixtureScrollHTML)
	defer env.Close()

	actions := []browseractions.Action{
		{Type: browseractions.ActionNavigate, URL: env.URL()},
		{Type: browseractions.ActionWait, For: browseractions.WaitLoad},
		{Type: browseractions.ActionScroll, Selector: "#target", Position: browseractions.ScrollTop},
		{Type: browseractions.ActionEvaluate, Expression: `() => {
			const r = document.querySelector('#target').getBoundingClientRect();
			return [Math.round(r.top), Math.round(r.bottom), window.innerHeight].join(',');
		}`},
		{Type: browseractions.ActionScroll, Selector: "#target", Position: browseractions.ScrollBottom},
		{Type: browseractions.ActionEvaluate, Expression: `() => {
			const r = document.querySelector('#target').getBoundingClientRect();
			return [Math.round(r.top), Math.round(r.bottom), window.innerHeight].join(',');
		}`},
	}

	result, err := Execute(context.Background(), env.page, actions, &ExecuteOptions{
		Validate:          true,
		ValidationProfile: browseractions.ValidationProfileStrict,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(result.Evaluations) != 2 {
		t.Fatalf("expected 2 evaluations, got %d", len(result.Evaluations))
	}

	topMetrics := mustCSVInts(t, result.Evaluations[0].Value, 3)
	bottomMetrics := mustCSVInts(t, result.Evaluations[1].Value, 3)

	if absInt(topMetrics[0]) > 12 {
		t.Fatalf("expected top alignment near 0, got %v", topMetrics)
	}
	if absInt(bottomMetrics[1]-bottomMetrics[2]) > 12 {
		t.Fatalf("expected bottom alignment near viewport bottom, got %v", bottomMetrics)
	}
}

func TestExecuteScreenshotModes(t *testing.T) {
	env := newRodTestEnv(t, fixtureFlowHTML)
	defer env.Close()

	cases := []struct {
		name              string
		actionFile        string
		writeFiles        bool
		includeInline     bool
		wantFileWritten   bool
		wantOutputFile    string
		wantOutputFileAbs bool
		wantData          bool
	}{
		{
			name:            "none",
			actionFile:      "",
			writeFiles:      false,
			includeInline:   false,
			wantFileWritten: false,
			wantOutputFile:  "",
			wantData:        false,
		},
		{
			name:              "file-only",
			actionFile:        "shots/file-only.png",
			writeFiles:        true,
			includeInline:     false,
			wantFileWritten:   true,
			wantOutputFile:    "shots/file-only.png",
			wantOutputFileAbs: true,
			wantData:          false,
		},
		{
			name:            "inline-only",
			actionFile:      "shots/inline-only.png",
			writeFiles:      false,
			includeInline:   true,
			wantFileWritten: false,
			wantOutputFile:  "shots/inline-only.png", // current behavior preserves requested path even when not written
			wantData:        true,
		},
		{
			name:              "both",
			actionFile:        "shots/both.png",
			writeFiles:        true,
			includeInline:     true,
			wantFileWritten:   true,
			wantOutputFile:    "shots/both.png",
			wantOutputFileAbs: true,
			wantData:          true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			actions := []browseractions.Action{
				{Type: browseractions.ActionNavigate, URL: env.URL()},
				{Type: browseractions.ActionWait, For: browseractions.WaitLoad},
				{Type: browseractions.ActionScreenshot, File: tc.actionFile},
			}
			result, err := Execute(context.Background(), env.page, actions, &ExecuteOptions{
				Validate:              true,
				ValidationProfile:     browseractions.ValidationProfileStrict,
				WriteFiles:            tc.writeFiles,
				IncludeScreenshotData: tc.includeInline,
				OutputDir:             tmpDir,
			})
			if err != nil {
				t.Fatalf("Execute returned error: %v", err)
			}
			if len(result.Screenshots) != 1 {
				t.Fatalf("expected 1 screenshot, got %d", len(result.Screenshots))
			}

			got := result.Screenshots[0]
			if tc.wantOutputFile == "" {
				if got.OutputFile != "" {
					t.Fatalf("expected empty OutputFile, got %q", got.OutputFile)
				}
			} else if tc.wantOutputFileAbs {
				want := filepath.Join(tmpDir, tc.wantOutputFile)
				if got.OutputFile != want {
					t.Fatalf("expected resolved output file %q, got %q", want, got.OutputFile)
				}
			} else if got.OutputFile != tc.wantOutputFile {
				t.Fatalf("expected output file %q, got %q", tc.wantOutputFile, got.OutputFile)
			}

			if tc.wantData {
				if got.Data == "" {
					t.Fatalf("expected inline screenshot data")
				}
				if _, err := base64.StdEncoding.DecodeString(got.Data); err != nil {
					t.Fatalf("invalid base64 screenshot data: %v", err)
				}
			} else if got.Data != "" {
				t.Fatalf("expected no inline screenshot data, got length=%d", len(got.Data))
			}

			if tc.actionFile != "" {
				abs := filepath.Join(tmpDir, tc.actionFile)
				_, statErr := os.Stat(abs)
				if tc.wantFileWritten && statErr != nil {
					t.Fatalf("expected screenshot file %q to exist: %v", abs, statErr)
				}
				if !tc.wantFileWritten && !errors.Is(statErr, os.ErrNotExist) {
					t.Fatalf("expected screenshot file %q not to exist, stat err=%v", abs, statErr)
				}
			}
		})
	}
}

func TestExecuteSleepHonorsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(30 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	result, err := Execute(ctx, nil, []browseractions.Action{
		{Type: browseractions.ActionSleep, Duration: 1000},
	}, nil)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if elapsed >= 900*time.Millisecond {
		t.Fatalf("expected cancellation before full sleep duration, elapsed=%v", elapsed)
	}
	if result.Failure == nil {
		t.Fatalf("expected failure in result")
	}
	if result.Failure.ActionIndex != 0 || result.Failure.ActionType != browseractions.ActionSleep {
		t.Fatalf("unexpected failure context: %+v", result.Failure)
	}
	if !strings.Contains(result.Failure.Message, "context canceled") {
		t.Fatalf("unexpected failure message: %+v", result.Failure)
	}
}

func TestExecuteErrorIncludesActionIndexAndType(t *testing.T) {
	result, err := Execute(context.Background(), nil, []browseractions.Action{
		{Type: browseractions.ActionType("bogus")},
	}, nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), `action 0 (bogus):`) {
		t.Fatalf("expected action context in error, got %q", err.Error())
	}
	if result.Failure == nil {
		t.Fatalf("expected result failure")
	}
	if result.Failure.ActionIndex != 0 || result.Failure.ActionType != browseractions.ActionType("bogus") {
		t.Fatalf("unexpected failure: %+v", result.Failure)
	}
}

type rodTestEnv struct {
	server  *httptest.Server
	browser *rod.Browser
	page    *rod.Page
}

func newRodTestEnv(t *testing.T, html string) *rodTestEnv {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping browser-backed rodexec test in short mode")
	}

	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(html))
	}))
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		srv.Close()
		t.Skipf("cannot bind local test server for rodexec integration tests: %v", err)
	}
	srv.Listener = ln
	srv.Start()

	bin, ok := findBrowserBinary()
	if !ok {
		srv.Close()
		t.Skip("no local chrome/chromium binary found for rodexec integration tests")
	}

	controlURL, err := newLauncher(RunOptions{
		Headless:   true,
		BinaryPath: bin,
	}).Launch()
	if err != nil {
		srv.Close()
		t.Skipf("failed to launch test browser: %v", err)
	}

	browser := rod.New().ControlURL(controlURL).MustConnect()
	page := browser.MustPage("")
	page.MustSetViewport(900, 420, 1.0, false)

	env := &rodTestEnv{server: srv, browser: browser, page: page}
	t.Cleanup(env.Close)
	return env
}

func (e *rodTestEnv) URL() string {
	return e.server.URL + "/"
}

func (e *rodTestEnv) Close() {
	if e == nil {
		return
	}
	if e.page != nil {
		e.page.MustClose()
		e.page = nil
	}
	if e.browser != nil {
		e.browser.MustClose()
		e.browser = nil
	}
	if e.server != nil {
		e.server.Close()
		e.server = nil
	}
}

func findBrowserBinary() (string, bool) {
	candidates := []string{"google-chrome", "chrome", "chromium", "chromium-browser"}
	for _, name := range candidates {
		if p, err := exec.LookPath(name); err == nil {
			return p, true
		}
	}
	return "", false
}

func mustAtoi(t *testing.T, s string) int {
	t.Helper()
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		t.Fatalf("failed to parse int from %q: %v", s, err)
	}
	return n
}

func mustCSVInts(t *testing.T, s string, want int) []int {
	t.Helper()
	raw := strings.Trim(strings.TrimSpace(s), `"`)
	parts := strings.Split(raw, ",")
	if len(parts) != want {
		t.Fatalf("expected %d csv values in %q, got %d", want, s, len(parts))
	}
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			t.Fatalf("failed parsing csv int from %q in %q: %v", p, s, err)
		}
		out = append(out, n)
	}
	return out
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

const fixtureFlowHTML = `<!doctype html>
<html>
  <body>
    <button id="show" type="button">Show</button>
    <div id="form" style="display:none;">
      <input id="name" type="text" />
      <button id="submit" type="button">Submit</button>
    </div>
    <div id="status" style="display:none;"></div>
    <script>
      const form = document.getElementById('form');
      const status = document.getElementById('status');
      document.getElementById('show').addEventListener('click', () => {
        form.style.display = 'block';
      });
      document.getElementById('submit').addEventListener('click', () => {
        const name = document.getElementById('name').value || 'anonymous';
        status.textContent = 'Saved for ' + name;
        status.style.display = 'block';
      });
    </script>
  </body>
</html>`

const fixtureScrollHTML = `<!doctype html>
<html>
  <body style="margin:0;">
    <div style="height:1400px;background:#f4f4f4;">top spacer</div>
    <div id="target" style="height:180px;background:#cde;border:1px solid #369;">target</div>
    <div style="height:800px;background:#f4f4f4;">bottom spacer</div>
  </body>
</html>`
