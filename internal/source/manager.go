package source

import (
	"context"
	"fmt"

	"bauer/internal/gdocs"
)

// Manager holds all registered source adapters and orchestrates fetching.
type Manager struct {
	credentialsPath string
}

// NewManager creates a Manager. credentialsPath is for Google Docs auth.
func NewManager(credentialsPath string) *Manager {
	return &Manager{credentialsPath: credentialsPath}
}

// Fetch runs the Google Docs adapter (and later, the Figma adapter when it exists).
// Returns a SourceBundle combining all source outputs.
func (m *Manager) Fetch(ctx context.Context, req Request) (*SourceBundle, error) {
	bundle := &SourceBundle{}

	if req.DocID != "" {
		gdocsClient, err := gdocs.NewClient(ctx, m.credentialsPath)
		if err != nil {
			return nil, fmt.Errorf("gdocs client init: %w", err)
		}
		result, err := gdocsClient.ProcessDocument(ctx, req.DocID)
		if err != nil {
			return nil, fmt.Errorf("gdocs fetch: %w", err)
		}
		bundle.Document = result
	}

	return bundle, nil
}
