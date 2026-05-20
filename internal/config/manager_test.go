package config_test

import (
	"os"
	"testing"

	"bauer/internal/config"
)

func TestEnvVarSource_OverridesDefault(t *testing.T) {
	t.Setenv("BAUER_DOC_ID", "env-doc-id")
	t.Setenv("BAUER_MODEL", "env-model")
	t.Setenv("BAUER_ARTIFACTS_DIR", "/tmp/env-artifacts")

	resolver := config.NewResolver(
		config.NewEnvVarSource(),
		config.NewDefaultsSource(),
	)
	cfg, err := resolver.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.DocID != "env-doc-id" {
		t.Errorf("DocID = %q, want %q", cfg.DocID, "env-doc-id")
	}
	if cfg.Model != "env-model" {
		t.Errorf("Model = %q, want %q", cfg.Model, "env-model")
	}
	if cfg.ArtifactsDir != "/tmp/env-artifacts" {
		t.Errorf("ArtifactsDir = %q, want %q", cfg.ArtifactsDir, "/tmp/env-artifacts")
	}
	// Default should still apply for unset fields
	if cfg.SummaryModel != "gpt-5-mini-high" {
		t.Errorf("SummaryModel = %q, want default %q", cfg.SummaryModel, "gpt-5-mini-high")
	}
}

func TestEnvVarSource_ZeroValueDoesNotOverride(t *testing.T) {
	// Clear any env vars that might interfere
	os.Unsetenv("BAUER_DOC_ID")
	os.Unsetenv("BAUER_MODEL")

	resolver := config.NewResolver(
		config.NewEnvVarSource(),
		config.NewDefaultsSource(),
	)
	cfg, err := resolver.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	// Default model should not be overridden by empty env var
	if cfg.Model != "gpt-5-mini-high" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-5-mini-high")
	}
	if cfg.ChunkSize != 1 {
		t.Errorf("ChunkSize = %d, want 1", cfg.ChunkSize)
	}
}

func TestEnvVarSource_BooleanFields(t *testing.T) {
	t.Setenv("BAUER_DRY_RUN", "true")
	t.Setenv("BAUER_PAGE_REFRESH", "false")

	src := config.NewEnvVarSource()
	cfg, err := src.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.DryRun == nil {
		t.Fatal("DryRun should not be nil when BAUER_DRY_RUN is set")
	}
	if !*cfg.DryRun {
		t.Errorf("DryRun = %v, want true", *cfg.DryRun)
	}

	if cfg.PageRefresh == nil {
		t.Fatal("PageRefresh should not be nil when BAUER_PAGE_REFRESH is set")
	}
	if *cfg.PageRefresh {
		t.Errorf("PageRefresh = %v, want false", *cfg.PageRefresh)
	}
}

func TestDefaultsSource_ProvidesSaneDefaults(t *testing.T) {
	src := config.NewDefaultsSource()
	cfg, err := src.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Model != "gpt-5-mini-high" {
		t.Errorf("Model = %q, want %q", cfg.Model, "gpt-5-mini-high")
	}
	if cfg.ChunkSize != 1 {
		t.Errorf("ChunkSize = %d, want 1", cfg.ChunkSize)
	}
	if cfg.ArtifactsDir != "./bauer-artifacts" {
		t.Errorf("ArtifactsDir = %q, want %q", cfg.ArtifactsDir, "./bauer-artifacts")
	}
	if cfg.DryRun == nil || *cfg.DryRun {
		t.Errorf("DryRun should be *false, got %v", cfg.DryRun)
	}
	if cfg.PageRefresh == nil || *cfg.PageRefresh {
		t.Errorf("PageRefresh should be *false, got %v", cfg.PageRefresh)
	}
}

func TestFlagsSource_OverridesEnvAndDefaults(t *testing.T) {
	t.Setenv("BAUER_DOC_ID", "env-doc")
	t.Setenv("BAUER_MODEL", "env-model")

	flags := config.CLIFlags{
		DocID: "flag-doc",
		Model: "flag-model",
	}

	resolver := config.NewResolver(
		config.NewFlagsSource(flags),
		config.NewEnvVarSource(),
		config.NewDefaultsSource(),
	)
	cfg, err := resolver.Resolve()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if cfg.DocID != "flag-doc" {
		t.Errorf("DocID = %q, want %q", cfg.DocID, "flag-doc")
	}
	if cfg.Model != "flag-model" {
		t.Errorf("Model = %q, want %q", cfg.Model, "flag-model")
	}
}

func TestResolver_CredentialsFallback(t *testing.T) {
	os.Unsetenv("BAUER_CREDENTIALS_PATH")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/gac/creds.json")

	src := config.NewEnvVarSource()
	cfg, err := src.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.CredentialsPath != "/gac/creds.json" {
		t.Errorf("CredentialsPath = %q, want %q", cfg.CredentialsPath, "/gac/creds.json")
	}
}

func TestResolver_BAUERCredentialsPathTakesPriority(t *testing.T) {
	t.Setenv("BAUER_CREDENTIALS_PATH", "/bauer/creds.json")
	t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/gac/creds.json")

	src := config.NewEnvVarSource()
	cfg, err := src.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.CredentialsPath != "/bauer/creds.json" {
		t.Errorf("CredentialsPath = %q, want BAUER_CREDENTIALS_PATH to win", cfg.CredentialsPath)
	}
}
