package workflow

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"bauer/internal/config"
	"bauer/internal/orchestrator"
)

// mockOrchestrator is a test double for orchestrator.Orchestrator. It records
// the config it was called with and returns canned results, so the workflow's
// parse phase can be exercised without touching Google Docs or the network.
type mockOrchestrator struct {
	result *orchestrator.OrchestrationResult
	err    error
	gotCfg *config.Config
}

func (m *mockOrchestrator) Execute(_ context.Context, cfg *config.Config) (*orchestrator.OrchestrationResult, error) {
	m.gotCfg = cfg
	return m.result, m.err
}

// parseResultWith builds an OrchestrationResult carrying a ParseResult with the
// given number of actionable suggestions.
func parseResultWith(suggestionCount int) *orchestrator.OrchestrationResult {
	suggestions := make([]orchestrator.SimplifiedActionableSuggestion, suggestionCount)
	for i := range suggestions {
		suggestions[i] = orchestrator.SimplifiedActionableSuggestion{ID: "suggestion"}
	}
	return &orchestrator.OrchestrationResult{
		ParseResult: &orchestrator.ParseResult{
			Metadata:              orchestrator.ParseResultMetadata{DocumentTitle: "Test Doc", DocumentID: "doc-123"},
			ActionableSuggestions: suggestions,
		},
		ParseOnly: true,
	}
}

func TestExecuteWorkflow_ParseOnly(t *testing.T) {
	outputDir := t.TempDir()
	orch := &mockOrchestrator{result: parseResultWith(2)}

	input := WorkflowInput{
		DocID:     "doc-123",
		OutputDir: outputDir,
		ParseOnly: true,
	}

	output, err := ExecuteWorkflow(context.Background(), input, orch)
	if err != nil {
		t.Fatalf("ExecuteWorkflow returned error: %v", err)
	}

	if output.Status != "success" {
		t.Errorf("Status = %q, want %q (errors: %v)", output.Status, "success", output.Errors)
	}

	if output.BauerResult.TotalSuggestions != 2 {
		t.Errorf("TotalSuggestions = %d, want 2", output.BauerResult.TotalSuggestions)
	}

	// The workflow must force parse-only on the orchestrator config regardless
	// of how the orchestrator is otherwise configured.
	if orch.gotCfg == nil || !orch.gotCfg.ParseOnly {
		t.Errorf("orchestrator was not invoked with ParseOnly=true; gotCfg=%+v", orch.gotCfg)
	}

	// The parse result file must be written and reported. outputDir comes from
	// t.TempDir(), which is already absolute, so it matches OutputFile directly.
	wantPath := filepath.Join(outputDir, "bauer-parse-result.json")
	if output.OutputFile != wantPath {
		t.Errorf("OutputFile = %q, want %q", output.OutputFile, wantPath)
	}

	data, statErr := os.ReadFile(wantPath)
	if statErr != nil {
		t.Fatalf("expected parse result file at %s: %v", wantPath, statErr)
	}

	var trimmed orchestrator.TrimmedParseResult
	if jsonErr := json.Unmarshal(data, &trimmed); jsonErr != nil {
		t.Fatalf("parse result file is not valid TrimmedParseResult JSON: %v", jsonErr)
	}
	if trimmed.DocumentTitle != "Test Doc" {
		t.Errorf("written DocumentTitle = %q, want %q", trimmed.DocumentTitle, "Test Doc")
	}
}

func TestExecuteWorkflow_OrchestratorError(t *testing.T) {
	orch := &mockOrchestrator{err: context.DeadlineExceeded}

	input := WorkflowInput{
		DocID:     "doc-123",
		OutputDir: t.TempDir(),
		ParseOnly: true,
	}

	output, err := ExecuteWorkflow(context.Background(), input, orch)
	if err == nil {
		t.Fatal("expected error from ExecuteWorkflow when orchestrator fails, got nil")
	}
	if output.Status != "failed" {
		t.Errorf("Status = %q, want %q", output.Status, "failed")
	}
	if len(output.Errors) == 0 {
		t.Error("expected output.Errors to be populated on failure")
	}
}

func TestDeriveCopilotBranchName(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"prefixed branch", "bauer/WD-123", "copilot/WD-123"},
		{"nested branch keeps remainder", "bauer/feature/login", "copilot/feature/login"},
		{"no slash", "bauer", "copilot/bauer"},
		{"leading slash", "/WD-123", "copilot/WD-123"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deriveCopilotBranchName(tt.in); got != tt.want {
				t.Errorf("deriveCopilotBranchName(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
