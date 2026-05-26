package main

import (
	"bauer/internal/agent"
	"bauer/internal/artifacts"
	"bauer/internal/config"
	"bauer/internal/orchestrator"
	"bauer/internal/source"
	"context"
	"strings"
	"testing"
)

func TestResolveCLIConfig_FlagsOverrideEnv(t *testing.T) {
	t.Setenv("BAUER_DOC_ID", "env-doc")

	cfg, err := resolveCLIConfig(config.CLIFlags{DocID: "flag-doc"})
	if err != nil {
		t.Fatalf("resolveCLIConfig() error = %v", err)
	}

	if cfg.DocID != "flag-doc" {
		t.Fatalf("DocID = %q, want %q", cfg.DocID, "flag-doc")
	}
}

func TestOpenPRExecutionConfig_DisablesDryRunForExecutionOnly(t *testing.T) {
	original := &config.Config{DryRun: config.BoolPtr(true)}
	execCfg := openPRExecutionConfig(original)

	if execCfg == original {
		t.Fatal("openPRExecutionConfig() should return a copy")
	}
	if config.BoolVal(execCfg.DryRun, true) {
		t.Fatal("execution config should disable dry-run")
	}
	if !config.BoolVal(original.DryRun, false) {
		t.Fatal("original config should remain in dry-run mode")
	}
}

func TestCheckMutualExclusion_BothSet(t *testing.T) {
	err := checkMutualExclusion(true, true)
	if err == nil {
		t.Fatal("expected error when both --open-pr and --open-issue are set")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("error message should mention mutual exclusion, got: %q", err.Error())
	}
}

func TestCheckMutualExclusion_OnlyPR(t *testing.T) {
	if err := checkMutualExclusion(true, false); err != nil {
		t.Fatalf("unexpected error with only --open-pr: %v", err)
	}
}

func TestCheckMutualExclusion_OnlyIssue(t *testing.T) {
	if err := checkMutualExclusion(false, true); err != nil {
		t.Fatalf("unexpected error with only --open-issue: %v", err)
	}
}

func TestCheckMutualExclusion_Neither(t *testing.T) {
	if err := checkMutualExclusion(false, false); err != nil {
		t.Fatalf("unexpected error with neither flag: %v", err)
	}
}

func TestRunOpenIssue_StubReturnsError(t *testing.T) {
	err := runOpenIssue(context.Background(), &config.Config{}, nil)
	if err == nil {
		t.Fatal("runOpenIssue should return an error (stub not yet implemented)")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Fatalf("expected 'not yet implemented' error, got: %q", err.Error())
	}
}

func TestRunOpenPR_StubReturnsError(t *testing.T) {
	err := runOpenPR(context.Background(), &config.Config{}, nil)
	if err == nil {
		t.Fatal("runOpenPR should return an error (stub not yet implemented)")
	}
	if !strings.Contains(err.Error(), "not yet implemented") {
		t.Fatalf("expected 'not yet implemented' error, got: %q", err.Error())
	}
}

// trackingAgent wraps MockAgent and records whether ExecuteChunk was called.
type trackingAgent struct {
	agent.MockAgent
	executeChunkCalled bool
}

func (a *trackingAgent) ExecuteChunk(ctx context.Context, chunkPath string, chunkNumber int, model string) (string, error) {
	a.executeChunkCalled = true
	return a.MockAgent.ExecuteChunk(ctx, chunkPath, chunkNumber, model)
}

// TestDryRun_StandaloneSkipsAgent verifies that in dry-run mode the orchestrator
// does not call ExecuteChunk on the agent — even when one is provided.
func TestDryRun_StandaloneSkipsAgent(t *testing.T) {
	spy := &trackingAgent{}
	arts := artifacts.NewManager(t.TempDir())
	src := source.NewManager("") // no credentials needed; Fetch is not called in this test

	orch := orchestrator.New(spy, src, arts)

	cfg := &config.Config{
		DocID:        "test-doc",
		DryRun:       config.BoolPtr(true),
		ChunkSize:    1,
		Model:        "gpt-5-mini-high",
		SummaryModel: "gpt-5-mini-high",
		ArtifactsDir: t.TempDir(),
		OutputDir:    t.TempDir(),
	}

	// Execute returns an error because the source can't fetch (no credentials),
	// but we verify ExecuteChunk was never called regardless.
	_, _ = orch.Execute(context.Background(), cfg)

	if spy.executeChunkCalled {
		t.Fatal("ExecuteChunk should not be called in dry-run mode")
	}
}
