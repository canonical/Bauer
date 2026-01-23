package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
)

// Config holds the runtime configuration for BAU.
type Config struct {
	// DocID is the Google Doc ID to extract feedback from.
	DocID string

	// CredentialsPath is the path to the Google Cloud service account JSON key file.
	CredentialsPath string

	// DryRun indicates if the tool should skip side-effect operations (Copilot CLI, PR creation).
	DryRun bool

	// ChunkSize is the maximum number of locations to process in a single chunk.
	// Default is 10 if not specified.
	ChunkSize int

	// OutputDir is the directory where generated prompt files will be saved.
	// Default is "bauer-output" if not specified.
	OutputDir string

	// Model is the Copilot model to use for sessions.
	// Default is "gpt-5-mini-high" if not specified.
	Model string
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
	chunkSize := flag.Int("chunk-size", 10, "Maximum number of locations per chunk (default: 10)")
	outputDir := flag.String("output-dir", "bauer-output", "Directory for generated prompt files (default: bauer-output)")
	model := flag.String("model", "gpt-5-mini-high", "Copilot model to use for sessions (default: gpt-5-mini-high)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExample:")
		fmt.Fprintf(os.Stderr, "  %s --doc-id \"1b9...\" --credentials ./creds.json --dry-run\n", os.Args[0])
	}

	flag.Parse()

	cfg := &Config{
		DocID:           *docID,
		CredentialsPath: *credentialsPath,
		DryRun:          *dryRun,
		ChunkSize:       *chunkSize,
		OutputDir:       *outputDir,
		Model:           *model,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.DocID == "" {
		return errors.New("missing required flag: --doc-id")
	}

	if c.CredentialsPath == "" {
		return errors.New("missing required flag: --credentials")
	}

	if c.ChunkSize <= 0 {
		return errors.New("chunk-size must be greater than 0")
	}

	// Verify credentials file exists
	info, err := os.Stat(c.CredentialsPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("credentials file not found: %s", c.CredentialsPath)
	}
	if err != nil {
		return fmt.Errorf("error checking credentials file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("credentials path is a directory, expected a file: %s", c.CredentialsPath)
	}

	return nil
}
