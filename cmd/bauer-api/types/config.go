package types

import (
	"bauer/internal/config"
	"bauer/internal/gdocs"
	"flag"
)

type APIConfig struct {
	// CredentialsJSON holds the raw service account JSON assembled from the
	// environment. It is used directly, avoiding any on-disk staging.
	CredentialsJSON []byte

	// OutputDir is the directory where generated prompt files will be saved.
	// Default is "bauer-output" if not specified.
	BaseOutputDir string

	// TargetRepo is the path (relative or absolute) to the target repository
	// where tasks should be executed. If not specified, uses the current directory.
	TargetRepo string `json:"target_repo"`
}

func LoadConfig() (*APIConfig, error) {
	baseOutputDir := flag.String("base-output-dir", "bauer-output", "Base path of directory for generated prompt files (default: bauer-output)")
	configFile := flag.String("config", "", "Path to JSON config file")
	targetRepo := flag.String("target-repo", "", "Path to target repository where tasks should be executed (default: current directory)")

	flag.Parse()

	// Credentials are always assembled from individual environment variables.
	credentialsJSON, err := gdocs.CredentialsFromEnv()
	if err != nil {
		return nil, err
	}

	if *configFile != "" {
		cfg, err := config.LoadFromJSONFile(*configFile)
		if err != nil {
			return nil, err
		}
		return &APIConfig{
			CredentialsJSON: credentialsJSON,
			BaseOutputDir:   cfg.OutputDir,
			TargetRepo:      cfg.TargetRepo,
		}, nil
	}

	cfg := &APIConfig{
		CredentialsJSON: credentialsJSON,
		BaseOutputDir:   *baseOutputDir,
		TargetRepo:      *targetRepo,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *APIConfig) Validate() error {
	return gdocs.ValidateCredentials(c.CredentialsJSON)
}
