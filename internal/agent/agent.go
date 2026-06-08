// internal/agent/agent.go
package agent

import "context"

// Agent is the interface any AI execution backend must implement.
// copilotcli.Client implements this; future backends (REST-based agents,
// test mocks, etc.) can implement it without touching the orchestrator.
type Agent interface {
	// Start boots the agent (e.g. starts the Copilot SDK server process).
	// Must be called before any other method. Callers should defer Stop().
	Start(ctx context.Context) error

	// ExecuteChunk sends a single chunk prompt file to the agent and returns
	// the full text output. chunkNum is for logging/display only.
	ExecuteChunk(ctx context.Context, chunkPath string, chunkNum int, model string) (string, error)

	// GenerateSummary produces a summary of all chunk outputs.
	// Only called when there are multiple chunks.
	GenerateSummary(ctx context.Context, outputs []string, model string) (string, error)

	// Stop shuts the agent down cleanly. Safe to call after a failed Start.
	Stop() error
}
