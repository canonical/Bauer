// Package agent provides test doubles for the orchestrator.Agent interface.
// The Agent interface itself is defined in the orchestrator package (at the
// consumer), following Go convention. This package exists so that tests in
// any package can construct a mock without copying test helpers.
//
// MockAgent satisfies orchestrator.Agent via structural typing — it does not
// import the orchestrator package, avoiding import cycles.
package agent

import "context"

// MockAgent is a test implementation of orchestrator.Agent where every method
// is a no-op returning nil. Set the Func fields to inject behaviour in tests.
type MockAgent struct {
	StartFunc           func(ctx context.Context) error
	ExecuteChunkFunc    func(ctx context.Context, chunkPath string, chunkNum int, model string) (string, error)
	GenerateSummaryFunc func(ctx context.Context, outputs []string, model string) (string, error)
	StopFunc            func() error

	// Call tracking
	StartCalls           int
	ExecuteChunkCalls    []ExecuteChunkCall
	GenerateSummaryCalls []GenerateSummaryCall
	StopCalls            int
}

// ExecuteChunkCall records a call to ExecuteChunk.
type ExecuteChunkCall struct {
	ChunkPath string
	ChunkNum  int
	Model     string
}

// GenerateSummaryCall records a call to GenerateSummary.
type GenerateSummaryCall struct {
	Outputs []string
	Model   string
}

// Start increments StartCalls and calls StartFunc if set.
func (m *MockAgent) Start(ctx context.Context) error {
	m.StartCalls++
	if m.StartFunc != nil {
		return m.StartFunc(ctx)
	}
	return nil
}

// ExecuteChunk records the call and calls ExecuteChunkFunc if set.
func (m *MockAgent) ExecuteChunk(ctx context.Context, chunkPath string, chunkNum int, model string) (string, error) {
	m.ExecuteChunkCalls = append(m.ExecuteChunkCalls, ExecuteChunkCall{
		ChunkPath: chunkPath,
		ChunkNum:  chunkNum,
		Model:     model,
	})
	if m.ExecuteChunkFunc != nil {
		return m.ExecuteChunkFunc(ctx, chunkPath, chunkNum, model)
	}
	return "", nil
}

// GenerateSummary records the call and calls GenerateSummaryFunc if set.
func (m *MockAgent) GenerateSummary(ctx context.Context, outputs []string, model string) (string, error) {
	copiedOutputs := append([]string(nil), outputs...)
	m.GenerateSummaryCalls = append(m.GenerateSummaryCalls, GenerateSummaryCall{
		Outputs: copiedOutputs,
		Model:   model,
	})
	if m.GenerateSummaryFunc != nil {
		return m.GenerateSummaryFunc(ctx, outputs, model)
	}
	return "", nil
}

// Stop increments StopCalls and calls StopFunc if set.
func (m *MockAgent) Stop() error {
	m.StopCalls++
	if m.StopFunc != nil {
		return m.StopFunc()
	}
	return nil
}
