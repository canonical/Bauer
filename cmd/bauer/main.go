package main

import (
	"bauer/internal/artifacts"
	"bauer/internal/config"
	"bauer/internal/copilotcli"
	"bauer/internal/orchestrator"
	"bauer/internal/source"
	"context"
	"flag"
	"fmt"
	"os"
)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)

	docID := fs.String("doc-id", "", "Google Doc ID to extract feedback from (required, or set BAUER_DOC_ID)")
	credentialsPath := fs.String("credentials", "", "Path to service account credentials JSON\n\t(falls back to BAUER_CREDENTIALS_PATH → GOOGLE_APPLICATION_CREDENTIALS → credentials.json)")
	chunkSize := fs.Int("chunk-size", 0, "Total number of chunks (default: 1, or 5 if --page-refresh)")
	pageRefresh := fs.Bool("page-refresh", false, "Use page-refresh-instructions template (default chunk-size: 5)")
	model := fs.String("model", "", "Copilot model for sessions (default: gpt-5-mini-high)")
	summaryModel := fs.String("summary-model", "", "Copilot model for summary session (default: gpt-5-mini-high)")
	dryRun := fs.Bool("dry-run", false, "In standalone mode: skip Copilot, write chunk files only.\n\tIn --open-pr mode: apply changes locally, skip PR creation.")
	artifactsDir := fs.String("artifacts-dir", "", "Directory for run artifacts (default: ./bauer-artifacts)")
	openPR := fs.Bool("open-pr", false, "Apply changes and open a pull request (mutually exclusive with --open-issue)")
	openIssue := fs.Bool("open-issue", false, "Generate a plan and open a GitHub issue without applying changes (mutually exclusive with --open-pr)")
	branchPrefix := fs.String("branch-prefix", "", "Prefix for created branches (default: bauer)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n\n")
		fmt.Fprintf(os.Stderr, "\t%s --doc-id <doc-id> [--credentials <path>] [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_DOC_ID                    Override for --doc-id\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_CREDENTIALS_PATH          Override for --credentials\n")
		fmt.Fprintf(os.Stderr, "\tGOOGLE_APPLICATION_CREDENTIALS  Fallback credentials path\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_MODEL                     Override for --model\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_SUMMARY_MODEL             Override for --summary-model\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_DRY_RUN                   Override for --dry-run (true/false)\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_ARTIFACTS_DIR             Override for --artifacts-dir\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_BRANCH_PREFIX             Override for --branch-prefix\n")
		fmt.Fprintf(os.Stderr, "\n")
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		if err == flag.ErrHelp {
			os.Exit(0)
		}
		os.Exit(2)
	}

	// Mutual exclusion check — before any network calls
	if err := checkMutualExclusion(*openPR, *openIssue); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Build CLIFlags — *bool fields are only set for explicitly-provided flags
	// so they don't override env vars when the user didn't pass the flag.
	flags := config.CLIFlags{
		DocID:           *docID,
		CredentialsPath: *credentialsPath,
		ChunkSize:       *chunkSize,
		Model:           *model,
		SummaryModel:    *summaryModel,
		ArtifactsDir:    *artifactsDir,
		BranchPrefix:    *branchPrefix,
	}
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "dry-run":
			flags.DryRun = config.BoolPtr(*dryRun)
		case "page-refresh":
			flags.PageRefresh = config.BoolPtr(*pageRefresh)
		case "open-pr":
			flags.OpenPR = config.BoolPtr(*openPR)
		case "open-issue":
			flags.OpenIssue = config.BoolPtr(*openIssue)
		}
	})

	cfg, err := resolveCLIConfig(flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: failed to get working directory:", err)
		os.Exit(1)
	}

	copilotAgent, err := copilotcli.NewClient(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: failed to create Copilot client:", err)
		os.Exit(1)
	}

	sources := source.NewManager(cfg.CredentialsPath)
	arts := artifacts.NewManager(cfg.ArtifactsDir)
	orch := orchestrator.New(copilotAgent, sources, arts)

	switch {
	case *openIssue:
		if err := runOpenIssue(ctx, cfg, orch); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case *openPR:
		if err := runOpenPR(ctx, cfg, orch); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		if _, err := orch.Execute(ctx, cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

// checkMutualExclusion returns an error if --open-pr and --open-issue are both set.
// Extracted for testability — main() calls os.Exit on error.
func checkMutualExclusion(openPR, openIssue bool) error {
	if openPR && openIssue {
		return fmt.Errorf("Error: --open-pr and --open-issue are mutually exclusive.\n  Use --open-pr to apply changes and open a PR.\n  Use --open-issue to generate a plan and open an issue without applying changes.")
	}
	return nil
}

// resolveCLIConfig builds a Config from CLI flags, falling back to environment variables
// and then hardcoded defaults. FlagsSource has highest priority.
func resolveCLIConfig(flags config.CLIFlags) (*config.Config, error) {
	return config.NewResolver(
		config.NewFlagsSource(flags),
		config.NewEnvVarSource(),
		config.NewDefaultsSource(),
	).Resolve()
}

// openPRExecutionConfig returns a copy of cfg with DryRun disabled.
// In --open-pr mode, Copilot runs to apply changes locally; only PR creation
// is skipped when the original cfg has DryRun=true.
func openPRExecutionConfig(original *config.Config) *config.Config {
	copy := *original
	copy.DryRun = config.BoolPtr(false)
	return &copy
}

// runOpenIssue is a stub — to be fully implemented in Phase 2.
func runOpenIssue(_ context.Context, _ *config.Config, _ orchestrator.Orchestrator) error {
	return fmt.Errorf("--open-issue not yet implemented")
}

// runOpenPR is a stub — to be fully implemented in Phase 2.
func runOpenPR(_ context.Context, _ *config.Config, _ orchestrator.Orchestrator) error {
	return fmt.Errorf("--open-pr not yet implemented")
}
