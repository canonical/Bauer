package gdocs

import (
	"context"
	"fmt"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

// Client holds the authenticated Google services.
type Client struct {
	Docs  *docs.Service
	Drive *drive.Service
}

// NewClient creates a new Google Docs and Drive client from the raw service
// account credentials JSON. Accepting the JSON directly (rather than a file
// path) lets callers supply credentials from an environment variable without
// staging them on disk.
func NewClient(ctx context.Context, credentialsJSON []byte) (*Client, error) {
	// Scopes for both Docs and Drive
	scopes := []string{
		"https://www.googleapis.com/auth/documents.readonly",
		"https://www.googleapis.com/auth/drive.readonly",
	}

	creds, err := google.CredentialsFromJSON(ctx, credentialsJSON, scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to parse service account credentials: %w", err)
	}

	// Initialize Docs service
	docsService, err := docs.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create docs service: %w", err)
	}

	// Initialize Drive service
	driveService, err := drive.NewService(ctx, option.WithCredentials(creds))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return &Client{
		Docs:  docsService,
		Drive: driveService,
	}, nil
}
