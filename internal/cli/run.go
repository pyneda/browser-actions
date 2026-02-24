package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pyneda/browser-actions/rodexec"
)

var runScript = rodexec.Run

func runCmd(ctx context.Context, args []string) int {
	fs := newFlagSet("run")
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON")
	timeout := fs.Duration("timeout", 30*time.Second, "Execution timeout")
	headed := fs.Bool("headed", false, "Run browser in headed mode")
	viewport := fs.String("viewport", "", "Viewport in WIDTHxHEIGHT format")
	proxy := fs.String("proxy", "", "HTTP proxy URL for browser")
	binary := fs.String("binary", "", "Browser binary path")
	profile := fs.String("profile", "strict", "Validation profile: strict|lenient")
	screenshotData := fs.String("screenshot-data", "none", "Screenshot data mode: none|inline")
	outputDir := fs.String("output-dir", "", "Base directory for screenshot files")
	format := fs.String("format", "auto", "Input format override: auto|json|yaml")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: browser-actions run [options] [FILE|-]")
		return 2
	}
	source := fs.Arg(0)

	vp, err := parseViewport(*viewport)
	if err != nil {
		if *jsonOut {
			printJSON(map[string]any{"ok": false, "error": err.Error()})
		} else {
			printCLIErrorHuman("run", "cli", source, err)
		}
		return 2
	}

	includeScreenshotData := false
	switch strings.ToLower(strings.TrimSpace(*screenshotData)) {
	case "", "none":
		includeScreenshotData = false
	case "inline":
		includeScreenshotData = true
	default:
		if *jsonOut {
			printJSON(map[string]any{"ok": false, "error": "invalid --screenshot-data (expected none|inline)"})
		} else {
			printCLIErrorHuman("run", "cli", source, fmt.Errorf("invalid --screenshot-data (expected none|inline)"))
		}
		return 2
	}

	script, err := readScript(source, *format)
	if err != nil {
		if *jsonOut {
			printJSON(map[string]any{
				"ok":          false,
				"stage":       "parse",
				"error":       err.Error(),
				"diagnostics": diagnosticsFromErr(err),
			})
		} else {
			printCLIErrorHuman("run", "parse", source, err)
		}
		return 1
	}

	runOpts := &rodexec.RunOptions{
		Headless:   !*headed,
		Timeout:    *timeout,
		Proxy:      *proxy,
		BinaryPath: *binary,
		Viewport:   vp,
		ExecuteOptions: &rodexec.ExecuteOptions{
			Validate:              true,
			ValidationProfile:     parseValidationProfile(*profile),
			IncludeScreenshotData: includeScreenshotData,
			WriteFiles:            true,
			OutputDir:             *outputDir,
		},
	}

	result, err := runScript(ctx, script, runOpts)
	if *jsonOut {
		out := map[string]any{
			"ok":     err == nil,
			"script": map[string]any{"title": script.Title, "actions": len(script.Actions)},
			"result": result,
		}
		if err != nil {
			out["error"] = err.Error()
			out["diagnostics"] = diagnosticsFromErr(err)
		}
		printJSON(out)
		if err != nil {
			return 1
		}
		return 0
	}

	printRunHuman(source, script, runOpts, result, err)
	if err != nil {
		return 1
	}
	return 0
}

func parseViewport(s string) (*rodexec.Viewport, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(strings.ToLower(s), "x")
	if len(parts) != 2 {
		return nil, fmt.Errorf("expected WIDTHxHEIGHT")
	}
	w, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil || w <= 0 {
		return nil, fmt.Errorf("invalid width")
	}
	h, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil || h <= 0 {
		return nil, fmt.Errorf("invalid height")
	}
	return &rodexec.Viewport{Width: w, Height: h}, nil
}
