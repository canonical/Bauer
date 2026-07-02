package config

import (
	"flag"
	"fmt"
	"os"
)

// Load parses command-line flags and returns a validated Config.
func Load() (*Config, error) {
	// Define flags
	// Note: We use a new FlagSet to facilitate testing if needed later,
	// but for now relying on the default flag set is sufficient for the main entry point.
	// To avoid conflicts if Load is called multiple times (e.g. in tests), we reset if needed,
	// but standard `flag` usage usually assumes run once per process.

	docID := flag.String("doc-id", "", "Google Doc ID to extract feedback from (required)")
	credentialsPath := flag.String("credentials", "", "Path to service account JSON (required)")
	configFile := flag.String("config", "", "Path to JSON config file")
	parseOnly := flag.Bool("parse-only", false, "Parse document to JSON only; skip GitHub integration")
	pageRefresh := flag.Bool("page-refresh", false, "Use page refresh mode with page-refresh-instructions template")
	outputDir := flag.String("output-dir", "bauer-output", "Directory for generated prompt files (default: bauer-output)")
	targetRepo := flag.String("target-repo", "", "Path to target repository where tasks should be executed (default: current directory)")

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
			{"--config", "<string>", "Path to JSON config file"},
			{"--doc-id", "<string>", "Google Doc ID to extract feedback from (required)"},
			{"--credentials", "<string>", "Path to service account JSON (required)"},
			{"--parse-only", "", "Parse document to JSON only; skip GitHub integration"},
			{"--page-refresh", "", "Use page refresh mode with page-refresh-instructions template"},
			{"--output-dir", "<string>", "Directory for generated prompt files (default: bauer-output)"},
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

	// If --config is provided, load from JSON file
	if *configFile != "" {
		return LoadFromJSONFile(*configFile)
	}

	// If no required flags are provided, show usage and exit
	if *docID == "" && *credentialsPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	cfg := &Config{
		DocID:           *docID,
		CredentialsPath: *credentialsPath,
		ParseOnly:       *parseOnly,
		PageRefresh:     *pageRefresh,
		OutputDir:       *outputDir,
		TargetRepo:      *targetRepo,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}
