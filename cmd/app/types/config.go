package types

import (
	"bauer/internal/config"
	"flag"
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
	credentialsPath := flag.String("credentials", "", "Path to service account JSON (required)")
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

	if *credentialsPath == "" {
		flag.Usage()
		os.Exit(1)
	}

	cfg := &APIConfig{
		CredentialsPath: *credentialsPath,
		BaseOutputDir:   *baseOutputDir,
		TargetRepo:      *targetRepo,
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *APIConfig) Validate() error {
	return config.ValidateCredentialsPath(c.CredentialsPath)
}
