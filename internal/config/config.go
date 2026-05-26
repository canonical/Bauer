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
	// Uses *bool so that an explicit false from CLI flags can override a true from a config file.
	DryRun *bool `json:"dry_run,omitempty"`

	// ChunkSize is the total number of chunks to create from all locations.
	// Default is 1 if not specified, or 5 if PageRefresh is true.
	ChunkSize int `json:"chunk_size"`

	// PageRefresh indicates if the page refresh mode should be used.
	// When true, uses page-refresh-instructions.md template and defaults ChunkSize to 5.
	// Uses *bool so that an explicit false from CLI flags can override a true from defaults.
	PageRefresh *bool `json:"page_refresh,omitempty"`

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

	// ArtifactsDir is the directory for run artifacts. Defaults to "./bauer-artifacts".
	// Overridden by --artifacts-dir flag or BAUER_ARTIFACTS_DIR env var.
	ArtifactsDir string `json:"artifacts_dir,omitempty"`

	// BranchPrefix is the prefix used when creating branches. Defaults to "bauer".
	BranchPrefix string `json:"branch_prefix,omitempty"`

	// FigmaURL is the Figma file URL for the design reference.
	FigmaURL string `json:"figma_url,omitempty"`

	// FigmaToken is the Figma API token. Overridden by BAUER_FIGMA_TOKEN env var.
	FigmaToken string `json:"figma_token,omitempty"`

	// GitHubRepo is the GitHub repository in owner/repo format.
	GitHubRepo string `json:"github_repo,omitempty"`

	// OpenPR controls whether a pull request is opened after applying changes.
	OpenPR *bool `json:"open_pr,omitempty"`

	// OpenIssue controls whether a GitHub issue is opened instead of a PR.
	OpenIssue *bool `json:"open_issue,omitempty"`
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
		c.ArtifactsDir = "./bauer-artifacts"
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

	if c.FigmaURL != "" && c.FigmaToken == "" {
		return errors.New("BAUER_FIGMA_TOKEN or FIGMA_TOKEN must be set when --figma-url is supplied")
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

// BoolPtr returns a pointer to the given bool value.
func BoolPtr(v bool) *bool { return &v }

// BoolVal dereferences a *bool pointer. Returns def if v is nil.
func BoolVal(v *bool, def bool) bool {
	if v == nil {
		return def
	}
	return *v
}
