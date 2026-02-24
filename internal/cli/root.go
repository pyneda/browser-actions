package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	browseractions "github.com/pyneda/browser-actions"
)

func Execute(ctx context.Context, args []string) int {
	if len(args) == 0 {
		printRootUsage(os.Stderr)
		return 2
	}

	switch args[0] {
	case "run":
		return runCmd(ctx, args[1:])
	case "validate":
		return validateCmd(args[1:])
	case "capabilities":
		return capabilitiesCmd(args[1:])
	case "version":
		return versionCmd(args[1:])
	case "-h", "--help", "help":
		printRootUsage(os.Stdout)
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printRootUsage(os.Stderr)
		return 2
	}
}

func printRootUsage(w io.Writer) {
	fmt.Fprintln(w, "browser-actions <command> [options] [FILE|-]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  run           Execute browser actions")
	fmt.Fprintln(w, "  validate      Validate browser actions file")
	fmt.Fprintln(w, "  capabilities  List supported actions and options")
	fmt.Fprintln(w, "  version       Print CLI version")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	fmt.Fprintln(w, "  browser-actions validate examples/basic-navigation.yaml")
	fmt.Fprintln(w, "  browser-actions run examples/dom-playground.yaml")
	fmt.Fprintln(w, "  browser-actions run examples/offline-local.yaml   # with local fixture server")
	fmt.Fprintln(w, "  browser-actions run --json examples/raw-array.json")
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

func readScript(path string, format string) (browseractions.BrowserActions, error) {
	opts := browseractions.ParseOptions{
		Format: parseInputFormat(format),
	}
	if path == "-" {
		return browseractions.LoadReader(os.Stdin, opts)
	}
	return browseractions.LoadFile(path, opts)
}

func parseInputFormat(s string) browseractions.InputFormat {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "", "auto":
		return browseractions.InputFormatAuto
	case "json":
		return browseractions.InputFormatJSON
	case "yaml", "yml":
		return browseractions.InputFormatYAML
	default:
		return browseractions.InputFormat(s)
	}
}

func parseValidationProfile(s string) browseractions.ValidationProfile {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case "", "strict":
		return browseractions.ValidationProfileStrict
	case "sukyan-legacy", "legacy":
		return browseractions.ValidationProfileLegacySukyan
	default:
		return browseractions.ValidationProfile(s)
	}
}

func printJSON(v any) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func diagnosticsFromErr(err error) []browseractions.Diagnostic {
	if de, ok := browseractions.AsDiagnosticsError(err); ok {
		return de.Diagnostics
	}
	if err == nil {
		return nil
	}
	return []browseractions.Diagnostic{
		{Kind: "error", Message: err.Error()},
	}
}
