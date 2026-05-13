package config

import (
	"fmt"
	"os"
	"strconv"
)

// Source provides a partial Config from a single input (env vars, flags, defaults).
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
// Typical order: NewEnvVarSource(), NewFlagsSource(flags), NewDefaultsSource()
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
	if override.DocID != "" {
		base.DocID = override.DocID
	}
	if override.CredentialsPath != "" {
		base.CredentialsPath = override.CredentialsPath
	}
	if override.Model != "" {
		base.Model = override.Model
	}
	if override.SummaryModel != "" {
		base.SummaryModel = override.SummaryModel
	}
	if override.OutputDir != "" {
		base.OutputDir = override.OutputDir
	}
	if override.BranchPrefix != "" {
		base.BranchPrefix = override.BranchPrefix
	}
	if override.ArtifactsDir != "" {
		base.ArtifactsDir = override.ArtifactsDir
	}
	if override.ChunkSize != 0 {
		base.ChunkSize = override.ChunkSize
	}
	if override.PageRefresh != nil {
		base.PageRefresh = override.PageRefresh
	}
	if override.DryRun != nil {
		base.DryRun = override.DryRun
	}
	if override.OpenPR != nil {
		base.OpenPR = override.OpenPR
	}
	if override.OpenIssue != nil {
		base.OpenIssue = override.OpenIssue
	}
	if override.FigmaURL != "" {
		base.FigmaURL = override.FigmaURL
	}
	if override.FigmaToken != "" {
		base.FigmaToken = override.FigmaToken
	}
	if override.TargetRepo != "" {
		base.TargetRepo = override.TargetRepo
	}
}

// BoolPtr returns a pointer to b. Use in Source.Load() to explicitly set a bool field.
func BoolPtr(b bool) *bool { return &b }

// BoolVal safely dereferences a *bool, returning def when the pointer is nil.
func BoolVal(p *bool, def bool) bool {
	if p == nil {
		return def
	}
	return *p
}

// EnvVarSource reads all BAUER_* env vars.
type EnvVarSource struct{}

func NewEnvVarSource() *EnvVarSource { return &EnvVarSource{} }

func (e *EnvVarSource) Load() (*Config, error) {
	cfg := &Config{}
	if v := os.Getenv("BAUER_CREDENTIALS_PATH"); v != "" {
		cfg.CredentialsPath = v
	} else if v := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"); v != "" {
		cfg.CredentialsPath = v
	}
	cfg.DocID = os.Getenv("BAUER_DOC_ID")
	cfg.Model = os.Getenv("BAUER_MODEL")
	cfg.SummaryModel = os.Getenv("BAUER_SUMMARY_MODEL")
	cfg.OutputDir = os.Getenv("BAUER_OUTPUT_DIR")
	cfg.BranchPrefix = os.Getenv("BAUER_BRANCH_PREFIX")
	cfg.ArtifactsDir = os.Getenv("BAUER_ARTIFACTS_DIR")
	cfg.FigmaURL = os.Getenv("BAUER_FIGMA_URL")
	cfg.FigmaToken = os.Getenv("BAUER_FIGMA_TOKEN")
	if cfg.FigmaToken == "" {
		cfg.FigmaToken = os.Getenv("FIGMA_TOKEN")
	}
	cfg.TargetRepo = os.Getenv("BAUER_TARGET_REPO")
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
	if v := os.Getenv("BAUER_OPEN_PR"); v != "" {
		b := v == "true"
		cfg.OpenPR = &b
	}
	if v := os.Getenv("BAUER_OPEN_ISSUE"); v != "" {
		b := v == "true"
		cfg.OpenIssue = &b
	}
	return cfg, nil
}

// DefaultsSource provides hardcoded fallback values.
type DefaultsSource struct{}

func NewDefaultsSource() *DefaultsSource { return &DefaultsSource{} }

func (d *DefaultsSource) Load() (*Config, error) {
	return &Config{
		Model:        "gpt-5-mini-high",
		SummaryModel: "gpt-5-mini-high",
		ChunkSize:    1,
		OutputDir:    "bauer-output",
		BranchPrefix: "bauer",
		ArtifactsDir: "bauer-artifacts",
		PageRefresh:  BoolPtr(false),
		DryRun:       BoolPtr(false),
	}, nil
}

// FlagsSource wraps parsed CLI flags.
type FlagsSource struct {
	flags CLIFlags
}

// CLIFlags holds all CLI flag values. Used by FlagsSource.
type CLIFlags struct {
	DocID           string
	CredentialsPath string
	DryRun          *bool
	ChunkSize       int
	PageRefresh     *bool
	OutputDir       string
	Model           string
	SummaryModel    string
	TargetRepo      string
	ArtifactsDir    string
	BranchPrefix    string
	OpenPR          *bool
	OpenIssue       *bool
	FigmaURL        string
}

func NewFlagsSource(f CLIFlags) *FlagsSource {
	return &FlagsSource{flags: f}
}

func (f *FlagsSource) Load() (*Config, error) {
	cfg := &Config{
		DocID:           f.flags.DocID,
		CredentialsPath: f.flags.CredentialsPath,
		ChunkSize:       f.flags.ChunkSize,
		OutputDir:       f.flags.OutputDir,
		Model:           f.flags.Model,
		SummaryModel:    f.flags.SummaryModel,
		TargetRepo:      f.flags.TargetRepo,
		ArtifactsDir:    f.flags.ArtifactsDir,
		BranchPrefix:    f.flags.BranchPrefix,
		FigmaURL:        f.flags.FigmaURL,
		DryRun:          f.flags.DryRun,
		PageRefresh:     f.flags.PageRefresh,
		OpenPR:          f.flags.OpenPR,
		OpenIssue:       f.flags.OpenIssue,
	}
	return cfg, nil
}

