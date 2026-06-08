// internal/agent/mock.go
package agent

import "context"

// MockAgent is a no-op Agent implementation for use in tests.
type MockAgent struct{}

func (m MockAgent) Start(_ context.Context) error { return nil }
func (m MockAgent) ExecuteChunk(_ context.Context, chunkPath string, _ int, _ string) (string, error) {
	return "mock output for chunk " + chunkPath, nil
}
func (m MockAgent) GenerateSummary(_ context.Context, _ []string, _ string) (string, error) {
	return "mock summary", nil
}
func (m MockAgent) Stop() error { return nil }
