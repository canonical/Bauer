package types

import (
	"bauer/internal/config"
	"flag"
	"fmt"
	"os"
)

type APIConfig struct {
	// OutputDir is the directory where generated prompt files will be saved.
	// Default is "bauer-output" if not specified.
	BaseOutputDir string

	// Model is the Copilot model to use for sessions.
	// Default is "gpt-5-mini-high" if not specified.
	Model string

	// SummaryModel is the Copilot model to use for the summary session.
	// Default is "gpt-5-mini-high" if not specified.
	SummaryModel string

	// TargetRepo is the path (relative or absolute) to the target repository
	// where tasks should be executed. If not specified, uses the current directory.
	TargetRepo string `json:"target_repo"`

	// APISecret is the shared secret used for API basic auth.
	APISecret string
}

func LoadConfig() (*APIConfig, error) {
	if err := config.LoadEnvFiles(); err != nil {
		return nil, err
	}

	baseOutputDir := flag.String("base-output-dir", config.DefaultOutputDir, "Base path of directory for generated prompt files (default: bauer-output)")
	model := flag.String("model", config.DefaultModel, "Copilot model to use for sessions (default: gpt-5-mini-high)")
	summaryModel := flag.String("summary-model", config.DefaultModel, "Copilot model to use for summary session (default: gpt-5-mini-high)")
	configFile := flag.String("config", "", "Path to JSON config file")
	targetRepo := flag.String("target-repo", "", "Path to target repository where tasks should be executed (default: current directory)")

	flag.Parse()
	apiCfg := newAPIConfig(*baseOutputDir, *model, *summaryModel, *targetRepo)

	if *configFile != "" {
		cfg, err := config.LoadFromJSONFile(*configFile)
		if err != nil {
			return nil, err
		}
		apiCfg = newAPIConfig(cfg.OutputDir, cfg.Model, cfg.SummaryModel, cfg.TargetRepo)
	}

	if err := apiCfg.Validate(); err != nil {
		return nil, err
	}

	return apiCfg, nil
}

func newAPIConfig(baseOutputDir, model, summaryModel, targetRepo string) *APIConfig {
	return &APIConfig{
		BaseOutputDir: baseOutputDir,
		Model:         model,
		SummaryModel:  summaryModel,
		TargetRepo:    targetRepo,
		APISecret:     os.Getenv("API_SECRET"),
	}
}

func (c *APIConfig) Validate() error {
	if err := config.ValidateCredentialsEnv(); err != nil {
		return err
	}
	if c.APISecret == "" {
		return fmt.Errorf("missing required environment variable: API_SECRET")
	}
	return nil
}
