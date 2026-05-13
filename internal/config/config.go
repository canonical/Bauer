package config

import (
	"bauer/internal/gdocs"
	"errors"
	"fmt"
	"os"
)

// Config holds the runtime configuration for BAU.
type Config struct {
	// DocID is the Google Doc ID to extract feedback from.
	DocID string `json:"doc_id"`

	// CredentialsPath is the path to the Google Cloud service account JSON key file.
	CredentialsPath string `json:"credentials"`

	// DryRun indicates if the tool should skip side-effect operations (Copilot CLI, PR creation).
	// Uses *bool so that explicit false overrides a default of true.
	DryRun *bool `json:"dry_run"`

	// ChunkSize is the total number of chunks to create from all locations.
	// Default is 1 if not specified, or 5 if PageRefresh is true.
	ChunkSize int `json:"chunk_size"`

	// PageRefresh indicates if the page refresh mode should be used.
	// When true, uses page-refresh-instructions.md template and defaults ChunkSize to 5.
	// Uses *bool so that explicit false overrides a default of true.
	PageRefresh *bool `json:"page_refresh"`

	// OutputDir is the directory where generated prompt files will be saved.
	// Default is "bauer-output" if not specified.
	OutputDir string `json:"output_dir"`

	// Model is the Copilot model to use for sessions.
	// Default is "gpt-5-mini-high" if not specified.
	Model string `json:"model"`

	// SummaryModel is the Copilot model to use for the summary session.
	// Default is "gpt-5-mini-high" if not specified.
	SummaryModel string `json:"summary_model"`

	// TargetRepo is the path (relative or absolute) to the target repository
	// where tasks should be executed. If not specified, uses the current directory.
	TargetRepo string `json:"target_repo"`

	// ArtifactsDir is the directory for append-only run artifacts.
	// Defaults to "./bauer-artifacts" if not specified.
	ArtifactsDir string `json:"artifacts_dir"`

	// BranchPrefix is the prefix for generated branch names.
	// Defaults to "bauer".
	BranchPrefix string `json:"branch_prefix"`

	// OpenPR indicates that after applying changes, a branch should be created
	// and a PR opened. Mutually exclusive with OpenIssue.
	OpenPR *bool `json:"open_pr"`

	// OpenIssue indicates that Copilot should be skipped and a GitHub issue
	// should be opened with the implementation plan. Mutually exclusive with OpenPR.
	OpenIssue *bool `json:"open_issue"`

	// FigmaURL is an optional Figma file URL for design context.
	FigmaURL string `json:"figma_url"`

	// FigmaToken is the Figma personal access token.
	FigmaToken string `json:"figma_token"`
}

// Apply default config values
func (c *Config) ApplyDefaults() {
	if c.ChunkSize == 0 {
		if BoolVal(c.PageRefresh, false) {
			c.ChunkSize = 5
		} else {
			c.ChunkSize = 1
		}
	}
	if c.OutputDir == "" {
		c.OutputDir = "bauer-output"
	}
	if c.Model == "" {
		c.Model = "gpt-5-mini-high"
	}
	if c.SummaryModel == "" {
		c.SummaryModel = "gpt-5-mini-high"
	}
	if c.ArtifactsDir == "" {
		c.ArtifactsDir = "bauer-artifacts"
	}
	if c.BranchPrefix == "" {
		c.BranchPrefix = "bauer"
	}
}

// Validate checks if the configuration is valid.
// It also applies default values for fields that are not set.
func (c *Config) Validate() error {
	// Apply defaults first
	c.ApplyDefaults()

	// Validate required fields
	if c.DocID == "" {
		return errors.New("missing required field: doc_id")
	}

	if c.ChunkSize <= 0 {
		return errors.New("chunk_size must be greater than 0")
	}

	return ValidateCredentialsPath(c.CredentialsPath)
}

func ValidateCredentialsPath(path string) error {
	// Verify credentials file exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("credentials file not found: %s", path)
	}
	if err != nil {
		return fmt.Errorf("error checking credentials file: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("credentials path is a directory, expected a file: %s", path)
	}

	// Validate credentials content
	if err := gdocs.ValidateCredentialsFile(path); err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}
