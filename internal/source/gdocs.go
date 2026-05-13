package source

import (
	"bauer/internal/gdocs"
	"context"
	"fmt"
)

// GDocsAdapter fetches extraction results from Google Docs.
// It creates a gdocs.Client per Fetch call because the credentials path
// varies per request (especially in the API server).
type GDocsAdapter struct{}

// NewGDocsAdapter creates a GDocsAdapter.
func NewGDocsAdapter() *GDocsAdapter {
	return &GDocsAdapter{}
}

// Name returns the adapter identifier.
func (a *GDocsAdapter) Name() string {
	return "gdocs"
}

// Fetch extracts suggestions from a Google Doc and returns a *gdocs.ProcessingResult.
// It expects req.CredentialsPath to be set; if empty, returns an error.
func (a *GDocsAdapter) Fetch(ctx context.Context, req Request) (any, error) {
	if req.DocID == "" {
		return nil, fmt.Errorf("gdocs: doc_id is required")
	}
	if req.CredentialsPath == "" {
		return nil, fmt.Errorf("gdocs: credentials_path is required")
	}

	client, err := gdocs.NewClient(ctx, req.CredentialsPath)
	if err != nil {
		return nil, fmt.Errorf("gdocs: failed to create client: %w", err)
	}

	return client.ProcessDocument(ctx, req.DocID)
}
