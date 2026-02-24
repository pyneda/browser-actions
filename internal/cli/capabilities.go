package cli

import browseractions "github.com/pyneda/browser-actions"

func capabilitiesCmd(args []string) int {
	fs := newFlagSet("capabilities")
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	caps := map[string]any{
		"backend": "rod",
		"actions": []string{
			string(browseractions.ActionNavigate),
			string(browseractions.ActionClick),
			string(browseractions.ActionFill),
			string(browseractions.ActionWait),
			string(browseractions.ActionAssert),
			string(browseractions.ActionScroll),
			string(browseractions.ActionScreenshot),
			string(browseractions.ActionSleep),
			string(browseractions.ActionEvaluate),
		},
		"wait_conditions": []string{
			string(browseractions.WaitVisible),
			string(browseractions.WaitHidden),
			string(browseractions.WaitEnabled),
			string(browseractions.WaitLoad),
		},
		"assert_conditions": []string{
			string(browseractions.AssertContains),
			string(browseractions.AssertEquals),
			string(browseractions.AssertVisible),
			string(browseractions.AssertHidden),
		},
		"scroll_positions": []string{
			string(browseractions.ScrollTop),
			string(browseractions.ScrollBottom),
		},
		"profiles": []string{
			string(browseractions.ValidationProfileStrict),
			string(browseractions.ValidationProfileLegacySukyan),
		},
		"formats": []string{
			string(browseractions.InputFormatAuto),
			string(browseractions.InputFormatJSON),
			string(browseractions.InputFormatYAML),
		},
	}

	if *jsonOut {
		printJSON(caps)
		return 0
	}

	printCapabilitiesHuman(caps)
	return 0
}
