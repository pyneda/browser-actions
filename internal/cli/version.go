package cli

var Version = "dev"

func versionCmd(args []string) int {
	fs := newFlagSet("version")
	jsonOut := fs.Bool("json", false, "Output machine-readable JSON")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *jsonOut {
		printJSON(map[string]any{"version": Version})
		return 0
	}
	printVersionHuman()
	return 0
}
