package cli

import (
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	browseractions "github.com/pyneda/browser-actions"
	"github.com/pyneda/browser-actions/rodexec"
)

type kvRow struct {
	Key   string
	Value string
}

func printHeading(w io.Writer, title string) {
	line := strings.Repeat("=", len(title))
	fmt.Fprintln(w, line)
	fmt.Fprintln(w, title)
	fmt.Fprintln(w, line)
}

func printSection(w io.Writer, title string) {
	fmt.Fprintln(w)
	fmt.Fprintln(w, title)
	fmt.Fprintln(w, strings.Repeat("-", len(title)))
}

func printKVTable(w io.Writer, rows []kvRow) {
	maxKey := 0
	for _, row := range rows {
		if len(row.Key) > maxKey {
			maxKey = len(row.Key)
		}
	}
	for _, row := range rows {
		fmt.Fprintf(w, "%-*s : %s\n", maxKey, row.Key, row.Value)
	}
}

func printDiagnosticsHuman(w io.Writer, diagnostics []browseractions.Diagnostic) {
	if len(diagnostics) == 0 {
		fmt.Fprintln(w, "(no diagnostics)")
		return
	}
	for i, d := range diagnostics {
		label := d.Kind
		if label == "" {
			label = "error"
		}
		path := d.Path
		if path == "" {
			path = "-"
		}
		code := d.Code
		if code == "" {
			code = "-"
		}
		fmt.Fprintf(w, "%2d. [%s] %s\n", i+1, strings.ToUpper(label), d.Message)
		fmt.Fprintf(w, "    path=%s code=%s\n", path, code)
	}
}

func printActionMix(w io.Writer, actions []browseractions.Action) {
	if len(actions) == 0 {
		fmt.Fprintln(w, "(no actions)")
		return
	}

	counts := map[browseractions.ActionType]int{}
	for _, a := range actions {
		counts[a.Type]++
	}

	ordered := []browseractions.ActionType{
		browseractions.ActionNavigate,
		browseractions.ActionClick,
		browseractions.ActionFill,
		browseractions.ActionWait,
		browseractions.ActionAssert,
		browseractions.ActionScroll,
		browseractions.ActionScreenshot,
		browseractions.ActionSleep,
		browseractions.ActionEvaluate,
	}

	first := true
	for _, t := range ordered {
		if counts[t] == 0 {
			continue
		}
		if !first {
			fmt.Fprint(w, ", ")
		}
		first = false
		fmt.Fprintf(w, "%s=%d", t, counts[t])
	}

	// Include any unknown types if validation was bypassed.
	var extras []string
	for t, n := range counts {
		known := false
		for _, k := range ordered {
			if t == k {
				known = true
				break
			}
		}
		if known {
			continue
		}
		extras = append(extras, fmt.Sprintf("%s=%d", t, n))
	}
	sort.Strings(extras)
	if len(extras) > 0 {
		if !first {
			fmt.Fprint(w, ", ")
		}
		fmt.Fprint(w, strings.Join(extras, ", "))
	}
	fmt.Fprintln(w)
}

func printRecentLogs(w io.Writer, logs []rodexec.LogEntry, max int) {
	if len(logs) == 0 {
		fmt.Fprintln(w, "(no logs)")
		return
	}
	start := 0
	if len(logs) > max {
		start = len(logs) - max
		fmt.Fprintf(w, "(showing last %d of %d logs)\n", len(logs)-start, len(logs))
	}
	for _, l := range logs[start:] {
		ts := "-"
		if !l.Timestamp.IsZero() {
			ts = l.Timestamp.Local().Format("15:04:05.000")
		}
		fmt.Fprintf(w, "%s %-5s %s\n", ts, l.Level, l.Text)
	}
}

func printArtifacts(w io.Writer, screenshots []rodexec.ScreenshotResult) {
	printed := false
	for i, s := range screenshots {
		if s.OutputFile == "" {
			continue
		}
		printed = true
		target := "page"
		if s.Selector != "" {
			target = s.Selector
		}
		fmt.Fprintf(w, "%2d. target=%s file=%s\n", i+1, target, s.OutputFile)
	}
	if !printed {
		fmt.Fprintln(w, "(no artifact files written)")
	}
}

func printRunHuman(
	source string,
	script browseractions.BrowserActions,
	runOpts *rodexec.RunOptions,
	result rodexec.ExecutionResult,
	runErr error,
) {
	printHeading(os.Stdout, "Browser Actions Run")

	mode := "headed"
	if runOpts != nil && runOpts.Headless {
		mode = "headless"
	}
	viewport := "-"
	if runOpts != nil && runOpts.Viewport != nil {
		viewport = fmt.Sprintf("%dx%d", runOpts.Viewport.Width, runOpts.Viewport.Height)
	}
	timeout := "-"
	if runOpts != nil && runOpts.Timeout > 0 {
		timeout = runOpts.Timeout.String()
	}
	profile := "-"
	if runOpts != nil && runOpts.ValidationProfile != "" {
		profile = string(runOpts.ValidationProfile)
	}

	printSection(os.Stdout, "Script")
	printKVTable(os.Stdout, []kvRow{
		{Key: "Title", Value: script.Title},
		{Key: "Source", Value: source},
		{Key: "Actions", Value: fmt.Sprintf("%d", len(script.Actions))},
		{Key: "Profile", Value: profile},
		{Key: "Backend", Value: "rod"},
		{Key: "Browser Mode", Value: mode},
		{Key: "Timeout", Value: timeout},
		{Key: "Viewport", Value: viewport},
	})
	// Action mix is a line, not a table row, to keep it readable.
	fmt.Printf("%-11s : ", "Action Mix")
	printActionMix(os.Stdout, script.Actions)

	printSection(os.Stdout, "Result")
	status := "FAILED"
	if runErr == nil {
		status = "OK"
	}
	printKVTable(os.Stdout, []kvRow{
		{Key: "Status", Value: status},
		{Key: "Completed", Value: fmt.Sprintf("%d/%d", result.CompletedActions, result.TotalActions)},
		{Key: "Duration", Value: result.Duration.Round(time.Millisecond).String()},
		{Key: "Screenshots", Value: fmt.Sprintf("%d", len(result.Screenshots))},
		{Key: "Evaluations", Value: fmt.Sprintf("%d", len(result.Evaluations))},
		{Key: "Logs", Value: fmt.Sprintf("%d", len(result.Logs))},
	})

	if result.Failure != nil {
		printSection(os.Stdout, "Failure")
		printKVTable(os.Stdout, []kvRow{
			{Key: "Action Index", Value: fmt.Sprintf("%d", result.Failure.ActionIndex)},
			{Key: "Action Type", Value: string(result.Failure.ActionType)},
			{Key: "Message", Value: result.Failure.Message},
		})
	}

	if len(result.Screenshots) > 0 {
		printSection(os.Stdout, "Artifacts")
		printArtifacts(os.Stdout, result.Screenshots)
	}

	if len(result.Evaluations) > 0 {
		printSection(os.Stdout, "Evaluations")
		for i, ev := range result.Evaluations {
			value := strings.TrimSpace(ev.Value)
			if value == "" {
				value = "(empty)"
			}
			if len(value) > 120 {
				value = value[:117] + "..."
			}
			fmt.Fprintf(os.Stdout, "%2d. %s\n", i+1, value)
		}
	}

	if runErr != nil || len(result.Logs) > 0 {
		printSection(os.Stdout, "Recent Logs")
		printRecentLogs(os.Stdout, result.Logs, 12)
	}

	if runErr != nil {
		if ds := diagnosticsFromErr(runErr); len(ds) > 0 {
			printSection(os.Stdout, "Diagnostics")
			printDiagnosticsHuman(os.Stdout, ds)
		}
		fmt.Fprintln(os.Stdout)
		fmt.Fprintf(os.Stdout, "Run error: %v\n", runErr)
	}
}

func printValidateHumanSuccess(script browseractions.BrowserActions, profile browseractions.ValidationProfile, source string) {
	printHeading(os.Stdout, "Browser Actions Validation")
	printSection(os.Stdout, "Summary")
	printKVTable(os.Stdout, []kvRow{
		{Key: "Status", Value: "VALID"},
		{Key: "Source", Value: source},
		{Key: "Title", Value: script.Title},
		{Key: "Actions", Value: fmt.Sprintf("%d", len(script.Actions))},
		{Key: "Profile", Value: string(profile)},
	})
	fmt.Printf("%-11s : ", "Action Mix")
	printActionMix(os.Stdout, script.Actions)
}

func printValidateHumanFailure(stage, source string, err error) {
	printHeading(os.Stderr, "Browser Actions Validation")
	printSection(os.Stderr, "Summary")
	printKVTable(os.Stderr, []kvRow{
		{Key: "Status", Value: "INVALID"},
		{Key: "Stage", Value: stage},
		{Key: "Source", Value: source},
		{Key: "Error", Value: err.Error()},
	})
	printSection(os.Stderr, "Diagnostics")
	printDiagnosticsHuman(os.Stderr, diagnosticsFromErr(err))
}

func printCLIErrorHuman(command, stage, source string, err error) {
	printHeading(os.Stderr, "browser-actions "+command)
	printSection(os.Stderr, "Error")
	printKVTable(os.Stderr, []kvRow{
		{Key: "Stage", Value: stage},
		{Key: "Source", Value: source},
		{Key: "Message", Value: err.Error()},
	})
	if ds := diagnosticsFromErr(err); len(ds) > 0 {
		printSection(os.Stderr, "Diagnostics")
		printDiagnosticsHuman(os.Stderr, ds)
	}
}

func printCapabilitiesHuman(caps map[string]any) {
	printHeading(os.Stdout, "Browser Actions Capabilities")
	printSection(os.Stdout, "Runtime")
	printKVTable(os.Stdout, []kvRow{
		{Key: "Backend", Value: fmt.Sprint(caps["backend"])},
	})

	printSection(os.Stdout, "Supported Actions")
	printStringList(os.Stdout, caps["actions"])

	printSection(os.Stdout, "Wait Conditions")
	printStringList(os.Stdout, caps["wait_conditions"])

	printSection(os.Stdout, "Assert Conditions")
	printStringList(os.Stdout, caps["assert_conditions"])

	printSection(os.Stdout, "Scroll Positions")
	printStringList(os.Stdout, caps["scroll_positions"])

	printSection(os.Stdout, "Validation Profiles")
	printStringList(os.Stdout, caps["profiles"])

	printSection(os.Stdout, "Input Formats")
	printStringList(os.Stdout, caps["formats"])
}

func printStringList(w io.Writer, raw any) {
	items, ok := raw.([]string)
	if !ok {
		fmt.Fprintln(w, fmt.Sprint(raw))
		return
	}
	for i, item := range items {
		fmt.Fprintf(w, "%2d. %s\n", i+1, item)
	}
}

func printVersionHuman() {
	printHeading(os.Stdout, "browser-actions")
	printKVTable(os.Stdout, []kvRow{
		{Key: "Version", Value: Version},
		{Key: "Backend", Value: "rod"},
	})
}
