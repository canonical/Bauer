package agent_test

import (
	"context"
	"testing"

	"bauer/internal/agent"
)

// Compile-time check that MockAgent implements Agent.
var _ agent.Agent = agent.MockAgent{}

func TestMockAgent_Start(t *testing.T) {
	a := agent.MockAgent{}
	if err := a.Start(context.Background()); err != nil {
		t.Fatalf("Start() = %v, want nil", err)
	}
}

func TestMockAgent_Stop(t *testing.T) {
	a := agent.MockAgent{}
	if err := a.Stop(); err != nil {
		t.Fatalf("Stop() = %v, want nil", err)
	}
}

func TestMockAgent_ExecuteChunk(t *testing.T) {
	a := agent.MockAgent{}
	out, err := a.ExecuteChunk(context.Background(), "chunk-1.md", 1, "gpt-4")
	if err != nil {
		t.Fatalf("ExecuteChunk() error = %v", err)
	}
	if out == "" {
		t.Fatal("ExecuteChunk() returned empty output")
	}
	if want := "mock output for chunk chunk-1.md"; out != want {
		t.Fatalf("ExecuteChunk() = %q, want %q", out, want)
	}
}

func TestMockAgent_GenerateSummary(t *testing.T) {
	a := agent.MockAgent{}
	summary, err := a.GenerateSummary(context.Background(), []string{"output1", "output2"}, "gpt-4")
	if err != nil {
		t.Fatalf("GenerateSummary() error = %v", err)
	}
	if want := "mock summary"; summary != want {
		t.Fatalf("GenerateSummary() = %q, want %q", summary, want)
	}
}
