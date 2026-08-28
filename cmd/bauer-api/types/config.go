package types

import (
	"bauer/internal/config"
	"bauer/internal/env"
	"flag"
	"fmt"
	"os"
)

type APIConfig struct {
	// CredentialsPath is the path to the Google Cloud service account JSON key file.
	CredentialsPath string

	// OutputDir is the directory where generated prompt files will be saved.
	// Default is "bauer-output" if not specified.
	BaseOutputDir string

	// TargetRepo is the path (relative or absolute) to the target repository
	// where tasks should be executed. If not specified, uses the current directory.
	TargetRepo string `json:"target_repo"`
}

func LoadConfig() (*APIConfig, error) {
	credentialsPath := flag.String("credentials", "", "Path to service account JSON (required unless APP_GOOGLE_CREDENTIALS is set)")
	baseOutputDir := flag.String("base-output-dir", "bauer-output", "Base path of directory for generated prompt files (default: bauer-output)")
	configFile := flag.String("config", "", "Path to JSON config file")
	targetRepo := flag.String("target-repo", "", "Path to target repository where tasks should be executed (default: current directory)")

	flag.Parse()

	if *configFile != "" {
		cfg, err := config.LoadFromJSONFile(*configFile)
		if err != nil {
			return nil, err
		}
		return &APIConfig{
			CredentialsPath: cfg.CredentialsPath,
			BaseOutputDir:   cfg.OutputDir,
			TargetRepo:      cfg.TargetRepo,
		}, nil
	}

	resolvedCredentialsPath := *credentialsPath

	// When no credentials file is provided via the flag, fall back to the raw
	// credentials JSON injected through the environment.
	if resolvedCredentialsPath == "" {
		if raw := env.GetGoEnv("GOOGLE_CREDENTIALS"); raw != "" {
			path, err := writeCredentialsToTempFile(raw)
			if err != nil {
				return nil, err
			}
			resolvedCredentialsPath = path
		}
	}

	if resolvedCredentialsPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	cfg := &APIConfig{
		CredentialsPath: resolvedCredentialsPath,
		BaseOutputDir:   *baseOutputDir,
		TargetRepo:      *targetRepo,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// writeCredentialsToTempFile persists raw service account JSON to a private
// temporary file and returns its path, allowing the file-based pipeline to
// consume credentials that were supplied through the environment.
func writeCredentialsToTempFile(content string) (string, error) {
	f, err := os.CreateTemp("", "bauer-credentials-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp credentials file: %w", err)
	}
	defer f.Close()

	if err := f.Chmod(0o600); err != nil {
		return "", fmt.Errorf("failed to secure temp credentials file: %w", err)
	}

	if _, err := f.WriteString(content); err != nil {
		return "", fmt.Errorf("failed to write temp credentials file: %w", err)
	}

	return f.Name(), nil
}

func (c *APIConfig) Validate() error {
	return config.ValidateCredentialsPath(c.CredentialsPath)
}
