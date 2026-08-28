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

	// CredentialsJSON holds the raw service account JSON. When set it takes
	// precedence over CredentialsPath, allowing credentials supplied through the
	// environment to be used without staging them on disk.
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

	// Prefer inline credentials JSON when provided.
	if len(c.CredentialsJSON) > 0 {
		return gdocs.ValidateCredentials(c.CredentialsJSON)
	}

	return ValidateCredentialsPath(c.CredentialsPath)
}

// ResolveCredentials returns the raw service account JSON, reading it from
// CredentialsPath when inline CredentialsJSON was not supplied.
func (c *Config) ResolveCredentials() ([]byte, error) {
	if len(c.CredentialsJSON) > 0 {
		return c.CredentialsJSON, nil
	}
	data, err := os.ReadFile(c.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}
	return data, nil
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
