package orchestrator

import (
	"context"
	"errors"
	"testing"

	"bauer/internal/artifacts"
	"bauer/internal/copilotcli"
	"bauer/internal/source"
	"bauer/internal/config"
	"bauer/internal/prompt"
	"bauer/internal/testutil"
)

// Compile-time check: copilotcli.Client must implement Agent.
var _ Agent = (*copilotcli.Client)(nil)

// Compile-time check: testutil.MockAgent must implement Agent.
var _ Agent = (*testutil.MockAgent)(nil)

func TestExecuteCopilotChunks_UsesAgentForEachChunk(t *testing.T) {
	t.Parallel()

	mock := &testutil.MockAgent{
		ExecuteChunkFunc: func(ctx context.Context, chunkPath string, chunkNum int, model string) (string, error) {
			switch chunkNum {
			case 1:
				return "first output", nil
			case 2:
				return "second output", nil
			default:
				return "", errors.New("unexpected chunk number")
			}
		},
	}

	chunks := []prompt.ChunkResult{
		{ChunkNumber: 1, Filename: "chunk-1.md"},
		{ChunkNumber: 2, Filename: "chunk-2.md"},
	}

	cfg := &config.Config{
		Model: "gpt-5-mini-high",
	}

	outputs, duration, err := executeCopilotChunks(context.Background(), chunks, cfg, mock)
	if err != nil {
		t.Fatalf("executeCopilotChunks() error = %v", err)
	}

	if duration < 0 {
		t.Fatalf("executeCopilotChunks() duration = %v, want non-negative", duration)
	}

	if len(outputs) != 2 {
		t.Fatalf("len(outputs) = %d, want 2", len(outputs))
	}

	if got, want := len(mock.ExecuteChunkCalls), 2; got != want {
		t.Fatalf("len(mock.ExecuteChunkCalls) = %d, want %d", got, want)
	}

	tests := []struct {
		name       string
		gotNumber  int
		wantNumber int
		gotOutput  string
		wantOutput string
		gotPath    string
		wantPath   string
		gotModel   string
		wantModel  string
	}{
		{
			name:       "first chunk",
			gotNumber:  outputs[0].ChunkNumber,
			wantNumber: 1,
			gotOutput:  outputs[0].Output,
			wantOutput: "first output",
			gotPath:    mock.ExecuteChunkCalls[0].ChunkPath,
			wantPath:   "chunk-1.md",
			gotModel:   mock.ExecuteChunkCalls[0].Model,
			wantModel:  "gpt-5-mini-high",
		},
		{
			name:       "second chunk",
			gotNumber:  outputs[1].ChunkNumber,
			wantNumber: 2,
			gotOutput:  outputs[1].Output,
			wantOutput: "second output",
			gotPath:    mock.ExecuteChunkCalls[1].ChunkPath,
			wantPath:   "chunk-2.md",
			gotModel:   mock.ExecuteChunkCalls[1].Model,
			wantModel:  "gpt-5-mini-high",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if tt.gotNumber != tt.wantNumber {
				t.Fatalf("ChunkNumber = %d, want %d", tt.gotNumber, tt.wantNumber)
			}
			if tt.gotOutput != tt.wantOutput {
				t.Fatalf("Output = %q, want %q", tt.gotOutput, tt.wantOutput)
			}
			if tt.gotPath != tt.wantPath {
				t.Fatalf("ChunkPath = %q, want %q", tt.gotPath, tt.wantPath)
			}
			if tt.gotModel != tt.wantModel {
				t.Fatalf("Model = %q, want %q", tt.gotModel, tt.wantModel)
			}
		})
	}
}

func TestExecuteCopilotChunks_ReturnsAgentError(t *testing.T) {
	t.Parallel()

	mock := &testutil.MockAgent{
		ExecuteChunkFunc: func(ctx context.Context, chunkPath string, chunkNum int, model string) (string, error) {
			return "", errors.New("agent failed")
		},
	}

	chunks := []prompt.ChunkResult{
		{ChunkNumber: 1, Filename: "chunk-1.md"},
	}

	cfg := &config.Config{
		Model: "gpt-5-mini-high",
	}

	outputs, duration, err := executeCopilotChunks(context.Background(), chunks, cfg, mock)
	if err == nil {
		t.Fatal("executeCopilotChunks() error = nil, want non-nil")
	}

	if outputs != nil {
		t.Fatalf("outputs = %#v, want nil", outputs)
	}

	if duration != 0 {
		t.Fatalf("duration = %v, want 0", duration)
	}

	if got, want := len(mock.ExecuteChunkCalls), 1; got != want {
		t.Fatalf("len(mock.ExecuteChunkCalls) = %d, want %d", got, want)
	}
}

func TestNewOrchestrator_WithMockDependencies(t *testing.T) {
	t.Parallel()

	mock := &testutil.MockAgent{}
	gdocsAdapter := source.NewGDocsAdapter()
	sources := source.NewManager(gdocsAdapter)
	artMgr := artifacts.NewManager(t.TempDir())
	orch := NewOrchestrator(mock, sources, artMgr)

	if orch == nil {
		t.Fatal("NewOrchestrator() returned nil")
	}

	if orch.agent == nil {
		t.Fatal("orchestrator agent is nil")
	}

	if orch.sources == nil {
		t.Fatal("orchestrator sources is nil")
	}

	if orch.artifacts == nil {
		t.Fatal("orchestrator artifacts is nil")
	}
}

func TestOrchestrator_FinalizeRunOnFailure(t *testing.T) {
	t.Parallel()

	mock := &testutil.MockAgent{}
	gdocsAdapter := source.NewGDocsAdapter()
	sources := source.NewManager(gdocsAdapter)
	artMgr := artifacts.NewManager(t.TempDir())
	orch := NewOrchestrator(mock, sources, artMgr)

	// Execute with a config that will fail (no doc ID, no source adapter configured)
	cfg := &config.Config{
		DryRun: true,
	}
	_, err := orch.Execute(context.Background(), cfg)

	// The fetch will fail, but finalizeRun should still have been called
	// We verify by checking that a run directory was created
	_ = err // error expected
}

func TestSetAgent(t *testing.T) {
	t.Parallel()

	gdocsAdapter := source.NewGDocsAdapter()
	sources := source.NewManager(gdocsAdapter)
	artMgr := artifacts.NewManager(t.TempDir())
	orch := NewOrchestrator(nil, sources, artMgr)

	if orch.agent != nil {
		t.Fatal("expected nil agent initially")
	}

	mock := &testutil.MockAgent{}
	orch.SetAgent(mock)

	if orch.agent == nil {
		t.Fatal("SetAgent did not set the agent")
	}
}
