package agent

import (
	"bauer/internal/orchestrator"
	"context"
	"errors"
	"testing"
)

// Compile-time check: MockAgent satisfies orchestrator.Agent.
// This lives in the test file to avoid an import cycle in production code.
var _ orchestrator.Agent = (*MockAgent)(nil)

func TestMockAgent_SatisfiesOrchestratorAgent(t *testing.T) {
	t.Parallel()

	mock := &MockAgent{}

	if err := mock.Start(context.Background()); err != nil {
		t.Fatalf("MockAgent.Start() returned error: %v", err)
	}
	if mock.StartCalls != 1 {
		t.Fatalf("StartCalls = %d, want 1", mock.StartCalls)
	}

	output, err := mock.ExecuteChunk(context.Background(), "test.md", 1, "test-model")
	if err != nil {
		t.Fatalf("MockAgent.ExecuteChunk() returned error: %v", err)
	}
	if output != "" {
		t.Fatalf("ExecuteChunk output = %q, want empty string", output)
	}
	if len(mock.ExecuteChunkCalls) != 1 {
		t.Fatalf("ExecuteChunkCalls length = %d, want 1", len(mock.ExecuteChunkCalls))
	}
	if mock.ExecuteChunkCalls[0].ChunkPath != "test.md" {
		t.Fatalf("ChunkPath = %q, want %q", mock.ExecuteChunkCalls[0].ChunkPath, "test.md")
	}
	if mock.ExecuteChunkCalls[0].ChunkNum != 1 {
		t.Fatalf("ChunkNum = %d, want 1", mock.ExecuteChunkCalls[0].ChunkNum)
	}
	if mock.ExecuteChunkCalls[0].Model != "test-model" {
		t.Fatalf("Model = %q, want %q", mock.ExecuteChunkCalls[0].Model, "test-model")
	}

	summary, err := mock.GenerateSummary(context.Background(), []string{"out1", "out2"}, "summary-model")
	if err != nil {
		t.Fatalf("MockAgent.GenerateSummary() returned error: %v", err)
	}
	if summary != "" {
		t.Fatalf("GenerateSummary output = %q, want empty string", summary)
	}
	if len(mock.GenerateSummaryCalls) != 1 {
		t.Fatalf("GenerateSummaryCalls length = %d, want 1", len(mock.GenerateSummaryCalls))
	}
	if len(mock.GenerateSummaryCalls[0].Outputs) != 2 {
		t.Fatalf("GenerateSummaryCalls Outputs length = %d, want 2", len(mock.GenerateSummaryCalls[0].Outputs))
	}
	if mock.GenerateSummaryCalls[0].Model != "summary-model" {
		t.Fatalf("Model = %q, want %q", mock.GenerateSummaryCalls[0].Model, "summary-model")
	}

	if err := mock.Stop(); err != nil {
		t.Fatalf("MockAgent.Stop() returned error: %v", err)
	}
	if mock.StopCalls != 1 {
		t.Fatalf("StopCalls = %d, want 1", mock.StopCalls)
	}
}

func TestMockAgent_CustomFuncs(t *testing.T) {
	t.Parallel()

	mock := &MockAgent{
		ExecuteChunkFunc: func(ctx context.Context, chunkPath string, chunkNum int, model string) (string, error) {
			return "custom output", nil
		},
		GenerateSummaryFunc: func(ctx context.Context, outputs []string, model string) (string, error) {
			return "custom summary", nil
		},
	}

	out, _ := mock.ExecuteChunk(context.Background(), "path.md", 1, "model")
	sum, _ := mock.GenerateSummary(context.Background(), []string{"a"}, "model")

	if out != "custom output" {
		t.Fatalf("ExecuteChunk output = %q, want %q", out, "custom output")
	}
	if sum != "custom summary" {
		t.Fatalf("GenerateSummary output = %q, want %q", sum, "custom summary")
	}
}

func TestMockAgent_StartError(t *testing.T) {
	t.Parallel()

	mock := &MockAgent{
		StartFunc: func(ctx context.Context) error {
			return errors.New("start failed")
		},
	}

	if err := mock.Start(context.Background()); err == nil {
		t.Fatal("expected error from StartFunc")
	}
}
