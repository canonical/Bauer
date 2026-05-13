package config

import (
	"flag"
	"fmt"
	"os"
)

// ParseCLIFlags parses command-line flags and returns a CLIFlags struct.
// It does NOT validate — validation happens after the config resolver merges
// all sources (flags, env vars, defaults).
func ParseCLIFlags(args []string) (CLIFlags, error) {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	var f CLIFlags

	fs.StringVar(&f.DocID, "doc-id", "", "Google Doc ID to extract feedback from (required)")
	fs.StringVar(&f.CredentialsPath, "credentials", "", "Path to service account JSON (required)")
	fs.StringVar(&f.OutputDir, "output-dir", "", "Directory for generated prompt files (default: bauer-output)")
	fs.StringVar(&f.Model, "model", "", "Copilot model to use for sessions (default: gpt-5-mini-high)")
	fs.StringVar(&f.SummaryModel, "summary-model", "", "Copilot model to use for summary session (default: gpt-5-mini-high)")
	fs.StringVar(&f.TargetRepo, "target-repo", "", "Path to target repository where tasks should be executed (default: current directory)")
	fs.StringVar(&f.ArtifactsDir, "artifacts-dir", "", "Directory for run artifacts (default: ./bauer-artifacts)")
	fs.StringVar(&f.BranchPrefix, "branch-prefix", "", "Branch name prefix for --open-pr (default: bauer)")
	fs.StringVar(&f.FigmaURL, "figma-url", "", "Figma file URL for design context (optional)")

	// Bool flags: we need to know if they were explicitly set.
	// We use custom bool vars that track whether the flag was seen.
	dryRunPtr := fs.Bool("dry-run", false, "Run extraction and planning only; skip Copilot and PR creation")
	pageRefreshPtr := fs.Bool("page-refresh", false, "Use page refresh mode with page-refresh-instructions template (default chunk size: 5)")
	openPRPtr := fs.Bool("open-pr", false, "After applying changes, create a branch and open a PR. Mutually exclusive with --open-issue.")
	openIssuePtr := fs.Bool("open-issue", false, "Skip Copilot, generate plan, open a GitHub issue instead. Mutually exclusive with --open-pr.")

	fs.IntVar(&f.ChunkSize, "chunk-size", 0, "Total number of chunks to create (default: 1, or 5 if --page-refresh is set)")

	// Custom usage message
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n\n")
		fmt.Fprintf(os.Stderr, "\t%s --doc-id <doc-id> --credentials <path> [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n\n")

		flags := []struct {
			name string
			typ  string
			desc string
		}{
			{"--doc-id", "<string>", "Google Doc ID to extract feedback from (required)"},
			{"--credentials", "<string>", "Path to service account JSON (required)"},
			{"--dry-run", "", "Run extraction and planning only; skip Copilot and PR creation"},
			{"--page-refresh", "", "Use page refresh mode with page-refresh-instructions template"},
			{"--chunk-size", "<int>", "Total number of chunks to create (default: 1, or 5 if --page-refresh is set)"},
			{"--output-dir", "<string>", "Directory for generated prompt files (default: bauer-output)"},
			{"--model", "<string>", "Copilot model to use for sessions (default: gpt-5-mini-high)"},
			{"--summary-model", "<string>", "Copilot model to use for summary session (default: gpt-5-mini-high)"},
			{"--target-repo", "<string>", "Path to target repository where tasks should be executed (default: current directory)"},
			{"--artifacts-dir", "<string>", "Directory for run artifacts (default: ./bauer-artifacts)"},
			{"--branch-prefix", "<string>", "Branch name prefix for --open-pr (default: bauer)"},
			{"--open-pr", "", "After applying changes, create a branch and open a PR"},
			{"--open-issue", "", "Skip Copilot, generate plan, open a GitHub issue"},
			{"--figma-url", "<string>", "Figma file URL for design context (optional)"},
		}

		for _, fl := range flags {
			if fl.typ != "" {
				fmt.Fprintf(os.Stderr, "\t%-25s %s\n", fl.name+" "+fl.typ, fl.desc)
			} else {
				fmt.Fprintf(os.Stderr, "\t%-25s %s\n", fl.name, fl.desc)
			}
		}

		fmt.Fprintf(os.Stderr, "\nUse \"%s --help\" to display this message.\n\n", os.Args[0])
	}

	if err := fs.Parse(args); err != nil {
		return CLIFlags{}, err
	}

	// Track whether bool flags were explicitly set
	setFlags := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) {
		setFlags[f.Name] = true
	})

	if setFlags["dry-run"] {
		f.DryRun = BoolPtr(*dryRunPtr)
	}
	if setFlags["page-refresh"] {
		f.PageRefresh = BoolPtr(*pageRefreshPtr)
	}
	if setFlags["open-pr"] {
		f.OpenPR = BoolPtr(*openPRPtr)
	}
	if setFlags["open-issue"] {
		f.OpenIssue = BoolPtr(*openIssuePtr)
	}

	return f, nil
}

// Load is the legacy entry point. It parses flags and returns a Config.
// Deprecated: use ParseCLIFlags + NewResolver for new code.
func Load() (*Config, error) {
	flags, err := ParseCLIFlags(os.Args[1:])
	if err != nil {
		return nil, err
	}

	cfg, err := NewResolver(
		NewEnvVarSource(),
		NewFlagsSource(flags),
		NewDefaultsSource(),
	).Resolve()
	if err != nil {
		return nil, err
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
