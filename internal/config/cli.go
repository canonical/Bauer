package config

import (
	"flag"
	"fmt"
	"os"
)

// CLIFlags holds the raw command-line flag values before environment-variable resolution.
type CLIFlags struct {
	DocID           string
	CredentialsPath string
	DryRun          *bool
	ChunkSize       int
	PageRefresh     *bool
	OutputDir       string
	Model           string
	SummaryModel    string
	TargetRepo      string
	ArtifactsDir    string
	BranchPrefix    string
	FigmaURL        string
	GitHubRepo      string
	OpenPR          *bool
	OpenIssue       *bool
}

// Load parses command-line flags and returns a validated Config.
func Load() (*Config, error) {
	// Define flags
	// Note: We use a new FlagSet to facilitate testing if needed later,
	// but for now relying on the default flag set is sufficient for the main entry point.
	// To avoid conflicts if Load is called multiple times (e.g. in tests), we reset if needed,
	// but standard `flag` usage usually assumes run once per process.

	docID := flag.String("doc-id", "", "Google Doc ID to extract feedback from (required)")
	credentialsPath := flag.String("credentials", "", "Path to service account JSON (required)")
	dryRun := flag.Bool("dry-run", false, "Run extraction and planning only; skip Copilot and PR creation")
	chunkSize := flag.Int("chunk-size", 0, "Total number of chunks to create (default: 1, or 5 if --page-refresh is set)")
	pageRefresh := flag.Bool("page-refresh", false, "Use page refresh mode with page-refresh-instructions template (default chunk size: 5)")
	outputDir := flag.String("output-dir", "bauer-output", "Directory for generated prompt files (default: bauer-output)")
	model := flag.String("model", "gpt-5-mini-high", "Copilot model to use for sessions (default: gpt-5-mini-high)")
	summaryModel := flag.String("summary-model", "gpt-5-mini-high", "Copilot model to use for summary session (default: gpt-5-mini-high)")
	targetRepo := flag.String("target-repo", "", "Path to target repository where tasks should be executed (default: current directory)")
	artifactsDir := flag.String("artifacts-dir", "", "Directory for run artifacts (default: ./bauer-artifacts)")
	figmaURL := flag.String("figma-url", "", "Figma file or design URL for design reference (optional)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n\n")
		fmt.Fprintf(os.Stderr, "\t%s --doc-id <doc-id> --credentials <path> [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n\n")

		// Manually format flags
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
			{"--artifacts-dir", "<string>", "Directory for run artifacts (default: ./bauer-artifacts)"},
			{"--figma-url", "<string>", "Figma file or design URL for design reference (optional)"},
			{"--model", "<string>", "Copilot model to use for sessions (default: gpt-5-mini-high)"},
			{"--summary-model", "<string>", "Copilot model to use for summary session (default: gpt-5-mini-high)"},
			{"--target-repo", "<string>", "Path to target repository where tasks should be executed (default: current directory)"},
		}

		for _, f := range flags {
			if f.typ != "" {
				fmt.Fprintf(os.Stderr, "\t%-25s %s\n", f.name+" "+f.typ, f.desc)
			} else {
				fmt.Fprintf(os.Stderr, "\t%-25s %s\n", f.name, f.desc)
			}
		}

		fmt.Fprintf(os.Stderr, "\nUse \"%s --help\" to display this message.\n\n", os.Args[0])
	}

	flag.Parse()

	// If no required flags are provided, show usage and exit
	if *docID == "" && *credentialsPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	cfg := &Config{
		DocID:           *docID,
		CredentialsPath: *credentialsPath,
		DryRun:          dryRun,
		ChunkSize:       *chunkSize,
		PageRefresh:     pageRefresh,
		OutputDir:       *outputDir,
		Model:           *model,
		SummaryModel:    *summaryModel,
		TargetRepo:      *targetRepo,
		ArtifactsDir:    *artifactsDir,
		FigmaURL:        *figmaURL,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
