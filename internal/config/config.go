package config

import (
	"bauer/internal/gdocs"
	"errors"
)

// Config holds the runtime configuration for BAU.
type Config struct {
	// DocID is the Google Doc ID to extract feedback from.
	DocID string `json:"doc_id"`

	// DryRun indicates if the tool should skip side-effect operations (Copilot CLI, PR creation).
	DryRun bool `json:"dry_run"`

	// ChunkSize is the total number of chunks to create from all locations.
	// Default is 1 if not specified, or 5 if PageRefresh is true.
	ChunkSize int `json:"chunk_size"`

	// PageRefresh indicates if the page refresh mode should be used.
	// When true, uses page-refresh-instructions.md template and defaults ChunkSize to 5.
	PageRefresh bool `json:"page_refresh"`

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
}

// Apply default config values
func (c *Config) ApplyDefaults() {
	if c.ChunkSize == 0 {
		if c.PageRefresh {
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

	if err := ValidateCredentialsEnv(); err != nil {
		return err
	}

	return nil
}

func ValidateCredentialsEnv() error {
	return gdocs.ValidateCredentialsEnv()
}
