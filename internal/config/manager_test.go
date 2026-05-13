package config

import (
	"os"
	"strings"
	"testing"
)

func TestResolver_Precedence(t *testing.T) {
	t.Run("flag overrides env var", func(t *testing.T) {
		t.Setenv("BAUER_DOC_ID", "env-doc")
		resolver := NewResolver(
			NewFlagsSource(CLIFlags{DocID: "flag-doc"}),
			NewEnvVarSource(),
			NewDefaultsSource(),
		)
		cfg, err := resolver.Resolve()
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if cfg.DocID != "flag-doc" {
			t.Errorf("DocID = %q, want %q", cfg.DocID, "flag-doc")
		}
	})

	t.Run("env var overrides default", func(t *testing.T) {
		t.Setenv("BAUER_MODEL", "env-model")
		resolver := NewResolver(
			NewEnvVarSource(),
			NewDefaultsSource(),
		)
		cfg, err := resolver.Resolve()
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if cfg.Model != "env-model" {
			t.Errorf("Model = %q, want %q", cfg.Model, "env-model")
		}
	})

	t.Run("flag overrides default", func(t *testing.T) {
		resolver := NewResolver(
			NewFlagsSource(CLIFlags{DocID: "flag-doc"}),
			NewDefaultsSource(),
		)
		cfg, err := resolver.Resolve()
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if cfg.DocID != "flag-doc" {
			t.Errorf("DocID = %q, want %q", cfg.DocID, "flag-doc")
		}
	})

	t.Run("zero value does not override lower-priority non-zero", func(t *testing.T) {
		resolver := NewResolver(
			NewFlagsSource(CLIFlags{DocID: ""}), // empty flag
			NewDefaultsSource(),
		)
		cfg, err := resolver.Resolve()
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		// DefaultsSource does not set DocID, so it stays empty
		if cfg.DocID != "" {
			t.Errorf("DocID = %q, want empty", cfg.DocID)
		}
	})

	t.Run("default values are applied", func(t *testing.T) {
		resolver := NewResolver(
			NewDefaultsSource(),
		)
		cfg, err := resolver.Resolve()
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if cfg.Model != "gpt-5-mini-high" {
			t.Errorf("Model = %q, want %q", cfg.Model, "gpt-5-mini-high")
		}
		if cfg.ChunkSize != 1 {
			t.Errorf("ChunkSize = %d, want %d", cfg.ChunkSize, 1)
		}
		if cfg.OutputDir != "bauer-output" {
			t.Errorf("OutputDir = %q, want %q", cfg.OutputDir, "bauer-output")
		}
		if cfg.BranchPrefix != "bauer" {
			t.Errorf("BranchPrefix = %q, want %q", cfg.BranchPrefix, "bauer")
		}
		if cfg.ArtifactsDir != "bauer-artifacts" {
			t.Errorf("ArtifactsDir = %q, want %q", cfg.ArtifactsDir, "bauer-artifacts")
		}
	})
}

func TestResolver_BooleanOverride(t *testing.T) {
	t.Run("explicit false flag overrides default true", func(t *testing.T) {
		// If defaults had DryRun=true (they don't, but testing override semantics)
		resolver := NewResolver(
			NewFlagsSource(CLIFlags{DryRun: BoolPtr(false)}),
			NewDefaultsSource(),
		)
		cfg, err := resolver.Resolve()
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if cfg.DryRun == nil || *cfg.DryRun != false {
			t.Errorf("DryRun = %v, want false", cfg.DryRun)
		}
	})

	t.Run("nil bool falls through to default", func(t *testing.T) {
		resolver := NewResolver(
			NewFlagsSource(CLIFlags{}), // no bool flags set
			NewDefaultsSource(),
		)
		cfg, err := resolver.Resolve()
		if err != nil {
			t.Fatalf("Resolve() error = %v", err)
		}
		if cfg.DryRun == nil || *cfg.DryRun != false {
			t.Errorf("DryRun = %v, want default false", cfg.DryRun)
		}
		if cfg.PageRefresh == nil || *cfg.PageRefresh != false {
			t.Errorf("PageRefresh = %v, want default false", cfg.PageRefresh)
		}
	})
}

func TestEnvVarSource_Load(t *testing.T) {
	t.Run("reads BAUER_* vars", func(t *testing.T) {
		t.Setenv("BAUER_DOC_ID", "my-doc")
		t.Setenv("BAUER_MODEL", "custom-model")
		t.Setenv("BAUER_CHUNK_SIZE", "3")
		t.Setenv("BAUER_DRY_RUN", "true")

		src := NewEnvVarSource()
		cfg, err := src.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.DocID != "my-doc" {
			t.Errorf("DocID = %q, want %q", cfg.DocID, "my-doc")
		}
		if cfg.Model != "custom-model" {
			t.Errorf("Model = %q, want %q", cfg.Model, "custom-model")
		}
		if cfg.ChunkSize != 3 {
			t.Errorf("ChunkSize = %d, want %d", cfg.ChunkSize, 3)
		}
		if cfg.DryRun == nil || *cfg.DryRun != true {
			t.Errorf("DryRun = %v, want true", cfg.DryRun)
		}
	})

	t.Run("GOOGLE_APPLICATION_CREDENTIALS fallback", func(t *testing.T) {
		os.Unsetenv("BAUER_CREDENTIALS_PATH")
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/fallback/creds.json")

		src := NewEnvVarSource()
		cfg, err := src.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.CredentialsPath != "/fallback/creds.json" {
			t.Errorf("CredentialsPath = %q, want %q", cfg.CredentialsPath, "/fallback/creds.json")
		}
	})

	t.Run("BAUER_CREDENTIALS_PATH takes priority over GOOGLE_APPLICATION_CREDENTIALS", func(t *testing.T) {
		t.Setenv("BAUER_CREDENTIALS_PATH", "/primary/creds.json")
		t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", "/fallback/creds.json")

		src := NewEnvVarSource()
		cfg, err := src.Load()
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}
		if cfg.CredentialsPath != "/primary/creds.json" {
			t.Errorf("CredentialsPath = %q, want %q", cfg.CredentialsPath, "/primary/creds.json")
		}
	})

	t.Run("ParseBool accepts TRUE and 1", func(t *testing.T) {
		for _, val := range []string{"TRUE", "1", "True", "true"} {
			t.Run(val, func(t *testing.T) {
				t.Setenv("BAUER_DRY_RUN", val)
				src := NewEnvVarSource()
				cfg, err := src.Load()
				if err != nil {
					t.Fatalf("Load() error = %v", err)
				}
				if cfg.DryRun == nil || *cfg.DryRun != true {
					t.Errorf("DryRun = %v, want true for %q", cfg.DryRun, val)
				}
			})
		}
	})

	t.Run("ParseBool rejects invalid values", func(t *testing.T) {
		t.Setenv("BAUER_DRY_RUN", "yes")
		src := NewEnvVarSource()
		_, err := src.Load()
		if err == nil {
			t.Fatal("expected error for invalid bool value")
		}
		if !strings.Contains(err.Error(), "BAUER_DRY_RUN") {
			t.Errorf("error should mention env var name: %v", err)
		}
	})
}

func TestFlagsSource_Load(t *testing.T) {
	src := NewFlagsSource(CLIFlags{
		DocID:           "flag-doc",
		CredentialsPath: "/flag/creds.json",
		DryRun:          BoolPtr(true),
		ChunkSize:       7,
	})
	cfg, err := src.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DocID != "flag-doc" {
		t.Errorf("DocID = %q, want %q", cfg.DocID, "flag-doc")
	}
	if cfg.CredentialsPath != "/flag/creds.json" {
		t.Errorf("CredentialsPath = %q, want %q", cfg.CredentialsPath, "/flag/creds.json")
	}
	if cfg.DryRun == nil || *cfg.DryRun != true {
		t.Errorf("DryRun = %v, want true", cfg.DryRun)
	}
	if cfg.ChunkSize != 7 {
		t.Errorf("ChunkSize = %d, want %d", cfg.ChunkSize, 7)
	}
}

func TestConfig_Validate_MissingFields(t *testing.T) {
	t.Run("missing doc-id", func(t *testing.T) {
		cfg := &Config{}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("expected error for missing doc-id")
		}
		if !strings.Contains(err.Error(), "doc_id") {
			t.Errorf("error message should mention doc_id: %v", err)
		}
	})

	t.Run("missing credentials", func(t *testing.T) {
		cfg := &Config{DocID: "my-doc"}
		err := cfg.Validate()
		if err == nil {
			t.Fatal("expected error for missing credentials")
		}
		if !strings.Contains(err.Error(), "credentials") {
			t.Errorf("error message should mention credentials: %v", err)
		}
	})
}
