package cli

import (
	"fmt"
	"os"

	browseractions "github.com/pyneda/browser-actions"
)

func validateCmd(args []string) int {
	fs := newFlagSet("validate")
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON")
	profile := fs.String("profile", "strict", "Validation profile: strict|sukyan-legacy")
	format := fs.String("format", "auto", "Input format override: auto|json|yaml")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(os.Stderr, "usage: browser-actions validate [options] [FILE|-]")
		return 2
	}
	source := fs.Arg(0)

	script, err := readScript(source, *format)
	if err != nil {
		if *jsonOut {
			printJSON(map[string]any{
				"valid":       false,
				"stage":       "parse",
				"error":       err.Error(),
				"diagnostics": diagnosticsFromErr(err),
			})
		} else {
			printValidateHumanFailure("parse", source, err)
		}
		return 1
	}

	profileValue := parseValidationProfile(*profile)
	err = browseractions.ValidateScript(script, profileValue)
	if *jsonOut {
		out := map[string]any{
			"valid":       err == nil,
			"title":       script.Title,
			"actions":     len(script.Actions),
			"diagnostics": diagnosticsFromErr(err),
		}
		if err != nil {
			out["error"] = err.Error()
		}
		printJSON(out)
		if err != nil {
			return 1
		}
		return 0
	}

	if err != nil {
		printValidateHumanFailure("validation", source, err)
		return 1
	}
	printValidateHumanSuccess(script, profileValue, source)
	return 0
}
