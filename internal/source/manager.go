package source

import (
	"bauer/internal/gdocs"
	"context"
	"fmt"
	"log/slog"
)

// Manager coordinates source adapters and produces a combined SourceBundle.
type Manager struct {
	adapters []Adapter
}

// NewManager creates a Manager with the given source adapters.
func NewManager(adapters ...Adapter) *Manager {
	return &Manager{adapters: adapters}
}

// Fetch calls each registered adapter and assembles the results into a SourceBundle.
// Adapters are called in order; later adapters may enrich (but not overwrite) data
// from earlier ones.
func (m *Manager) Fetch(ctx context.Context, req Request) (*SourceBundle, error) {
	bundle := &SourceBundle{}

	for _, a := range m.adapters {
		slog.Info("Fetching source", "adapter", a.Name())

		result, err := a.Fetch(ctx, req)
		if err != nil {
			return nil, fmt.Errorf("source %s: %w", a.Name(), err)
		}

		// Assign to the correct bundle field based on adapter type.
		// This is the one place where the manager knows about specific source types;
		// it must be updated when new source types are added.
		switch v := result.(type) {
		case *gdocs.ProcessingResult:
			bundle.Document = v
		default:
			slog.Warn("Unknown source result type", "adapter", a.Name(), "type", fmt.Sprintf("%T", v))
		}
	}

	return bundle, nil
}
