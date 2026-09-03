package config

import (
	"bauer/internal/gdocs"
	"errors"
)

// Config holds the runtime configuration for BAU.
type Config struct {
	// DocID is the Google Doc ID to extract feedback from.
	DocID string `json:"doc_id"`

	// CredentialsJSON holds the raw service account JSON assembled from the
	// environment. Credentials are never read from disk.
	CredentialsJSON []byte `json:"-"`

	// ParseOnly indicates Phase 1 mode - parse document only, skip GitHub integration
	ParseOnly bool `json:"parse_only"`

	// PageRefresh indicates if the page refresh mode should be used.
	// When true, uses page-refresh-instructions.md template.
	PageRefresh bool `json:"page_refresh"`

	// OutputDir is the directory where generated prompt files will be saved.
	// Default is "bauer-output" if not specified.
	OutputDir string `json:"output_dir"`

	// TargetRepo is the path (relative or absolute) to the target repository
	// where tasks should be executed. If not specified, uses the current directory.
	TargetRepo string `json:"target_repo"`
}

// Apply default config values
func (c *Config) ApplyDefaults() {
	if c.OutputDir == "" {
		c.OutputDir = "bauer-output"
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

	// Credentials are assembled from the environment. Validate them only when
	// present; entry points build and validate them before use.
	if len(c.CredentialsJSON) > 0 {
		return gdocs.ValidateCredentials(c.CredentialsJSON)
	}

	return nil
}

// ResolveCredentials returns the raw service account JSON assembled from the
// environment.
func (c *Config) ResolveCredentials() ([]byte, error) {
	if len(c.CredentialsJSON) == 0 {
		return nil, errors.New("missing service account credentials")
	}
	return c.CredentialsJSON, nil
}
