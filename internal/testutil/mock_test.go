package testutil

import (
	"bauer/internal/orchestrator"
	"context"
	"errors"
	"testing"
)

// Compile-time check: MockAgent must implement orchestrator.Agent.
var _ orchestrator.Agent = (*MockAgent)(nil)

func TestMockAgent_SatisfiesOrchestratorAgent(t *testing.T) {
	t.Parallel()

	mock := &MockAgent{}

	if err := mock.Start(context.Background()); err != nil {
		t.Fatalf("MockAgent.Start() error: %v", err)
	}
	if mock.StartCalls != 1 {
		t.Fatalf("StartCalls = %d, want 1", mock.StartCalls)
	}

	output, err := mock.ExecuteChunk(context.Background(), "test.md", 1, "model")
	if err != nil {
		t.Fatalf("ExecuteChunk error: %v", err)
	}
	if output != "" {
		t.Fatalf("ExecuteChunk output = %q, want empty", output)
	}
	if len(mock.ExecuteChunkCalls) != 1 {
		t.Fatalf("ExecuteChunkCalls = %d, want 1", len(mock.ExecuteChunkCalls))
	}

	summary, err := mock.GenerateSummary(context.Background(), []string{"a"}, "model")
	if err != nil {
		t.Fatalf("GenerateSummary error: %v", err)
	}
	if summary != "" {
		t.Fatalf("GenerateSummary output = %q, want empty", summary)
	}

	if err := mock.Stop(); err != nil {
		t.Fatalf("Stop error: %v", err)
	}
	if mock.StopCalls != 1 {
		t.Fatalf("StopCalls = %d, want 1", mock.StopCalls)
	}
}

func TestMockAgent_CustomFuncs(t *testing.T) {
	t.Parallel()

	mock := &MockAgent{
		ExecuteChunkFunc: func(_ context.Context, _ string, _ int, _ string) (string, error) {
			return "custom", nil
		},
		GenerateSummaryFunc: func(_ context.Context, _ []string, _ string) (string, error) {
			return "summary", nil
		},
		StartFunc: func(_ context.Context) error { return errors.New("start fail") },
	}

	if err := mock.Start(context.Background()); err == nil {
		t.Fatal("expected Start error")
	}

	out, _ := mock.ExecuteChunk(context.Background(), "p", 1, "m")
	if out != "custom" {
		t.Fatalf("ExecuteChunk = %q, want %q", out, "custom")
	}

	sum, _ := mock.GenerateSummary(context.Background(), nil, "m")
	if sum != "summary" {
		t.Fatalf("GenerateSummary = %q, want %q", sum, "summary")
	}
}
