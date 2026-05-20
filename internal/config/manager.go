package config

import (
	"fmt"
	"os"
	"strconv"
)

// Source provides a partial Config from a single input.
// Fields not provided by this source should be zero-valued.
type Source interface {
	Load() (*Config, error)
}

// Resolver merges multiple Sources in priority order.
// First source listed = highest priority.
type Resolver struct {
	sources []Source
}

// NewResolver creates a Resolver. List sources highest-priority first.
func NewResolver(sources ...Source) *Resolver {
	return &Resolver{sources: sources}
}

// Resolve merges all sources and returns the final Config.
func (r *Resolver) Resolve() (*Config, error) {
	result := &Config{}
	// Apply sources lowest-priority first, higher priority overwrites
	for i := len(r.sources) - 1; i >= 0; i-- {
		partial, err := r.sources[i].Load()
		if err != nil {
			return nil, fmt.Errorf("config source %d: %w", i, err)
		}
		mergeConfig(result, partial)
	}
	return result, nil
}

func mergeConfig(base, override *Config) {
	if override.DocID != ""           { base.DocID = override.DocID }
	if override.CredentialsPath != "" { base.CredentialsPath = override.CredentialsPath }
	if override.Model != ""           { base.Model = override.Model }
	if override.SummaryModel != ""    { base.SummaryModel = override.SummaryModel }
	if override.ArtifactsDir != ""    { base.ArtifactsDir = override.ArtifactsDir }
	if override.BranchPrefix != ""    { base.BranchPrefix = override.BranchPrefix }
	if override.ChunkSize != 0        { base.ChunkSize = override.ChunkSize }
	if override.GitHubRepo != ""      { base.GitHubRepo = override.GitHubRepo }
	if override.FigmaURL != ""        { base.FigmaURL = override.FigmaURL }
	if override.FigmaToken != ""      { base.FigmaToken = override.FigmaToken }
	if override.OutputDir != ""       { base.OutputDir = override.OutputDir }
	if override.TargetRepo != ""      { base.TargetRepo = override.TargetRepo }
	if override.PageRefresh != nil    { base.PageRefresh = override.PageRefresh }
	if override.DryRun != nil         { base.DryRun = override.DryRun }
	if override.OpenPR != nil         { base.OpenPR = override.OpenPR }
	if override.OpenIssue != nil      { base.OpenIssue = override.OpenIssue }
}

// EnvVarSource reads BAUER_* env vars.
type EnvVarSource struct{}

func NewEnvVarSource() *EnvVarSource { return &EnvVarSource{} }

func (e *EnvVarSource) Load() (*Config, error) {
	cfg := &Config{}
	// Credentials — check BAUER_CREDENTIALS_PATH, then GOOGLE_APPLICATION_CREDENTIALS
	if v := os.Getenv("BAUER_CREDENTIALS_PATH"); v != "" {
		cfg.CredentialsPath = v
	} else if v := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); v != "" {
		cfg.CredentialsPath = v
	}
	cfg.DocID        = os.Getenv("BAUER_DOC_ID")
	cfg.Model        = os.Getenv("BAUER_MODEL")
	cfg.SummaryModel = os.Getenv("BAUER_SUMMARY_MODEL")
	cfg.ArtifactsDir = os.Getenv("BAUER_ARTIFACTS_DIR")
	cfg.BranchPrefix = os.Getenv("BAUER_BRANCH_PREFIX")
	cfg.GitHubRepo   = os.Getenv("BAUER_GITHUB_REPO")
	cfg.FigmaURL     = os.Getenv("BAUER_FIGMA_URL")
	cfg.FigmaToken   = os.Getenv("BAUER_FIGMA_TOKEN")
	if cfg.FigmaToken == "" {
		cfg.FigmaToken = os.Getenv("FIGMA_TOKEN")
	}
	if v := os.Getenv("BAUER_CHUNK_SIZE"); v != "" {
		cfg.ChunkSize, _ = strconv.Atoi(v)
	}
	if v := os.Getenv("BAUER_PAGE_REFRESH"); v != "" {
		b := v == "true"
		cfg.PageRefresh = &b
	}
	if v := os.Getenv("BAUER_DRY_RUN"); v != "" {
		b := v == "true"
		cfg.DryRun = &b
	}
	return cfg, nil
}

// DefaultsSource provides hardcoded fallback values.
type DefaultsSource struct{}

func NewDefaultsSource() *DefaultsSource { return &DefaultsSource{} }

func (d *DefaultsSource) Load() (*Config, error) {
	return &Config{
		Model:           "gpt-5-mini-high",
		SummaryModel:    "gpt-5-mini-high",
		ChunkSize:       1,
		ArtifactsDir:    "./bauer-artifacts",
		BranchPrefix:    "bauer",
		CredentialsPath: "credentials.json",
		PageRefresh:     BoolPtr(false),
		DryRun:          BoolPtr(false),
		OpenPR:          BoolPtr(false),
		OpenIssue:       BoolPtr(false),
	}, nil
}

// FlagsSource converts a CLIFlags struct into a Config.
// Only non-zero values are included so they properly override lower-priority sources.
type FlagsSource struct {
	flags CLIFlags
}

// NewFlagsSource creates a FlagsSource from the given CLI flags.
func NewFlagsSource(flags CLIFlags) *FlagsSource {
	return &FlagsSource{flags: flags}
}

func (f *FlagsSource) Load() (*Config, error) {
	return &Config{
		DocID:           f.flags.DocID,
		CredentialsPath: f.flags.CredentialsPath,
		DryRun:          f.flags.DryRun,
		ChunkSize:       f.flags.ChunkSize,
		PageRefresh:     f.flags.PageRefresh,
		OutputDir:       f.flags.OutputDir,
		Model:           f.flags.Model,
		SummaryModel:    f.flags.SummaryModel,
		TargetRepo:      f.flags.TargetRepo,
		ArtifactsDir:    f.flags.ArtifactsDir,
		BranchPrefix:    f.flags.BranchPrefix,
		GitHubRepo:      f.flags.GitHubRepo,
		OpenPR:          f.flags.OpenPR,
		OpenIssue:       f.flags.OpenIssue,
	}, nil
}
