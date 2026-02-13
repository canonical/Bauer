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

// NewClient creates a new Google Docs and Drive client using service account credentials from environment variables.
func NewClient(ctx context.Context) (*Client, error) {
	credentials, err := LoadCredentialsFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load service account credentials from environment: %w", err)
	}

	credentialsJSON, err := credentials.JSON()
	if err != nil {
		return nil, err
	}

	// Scopes for both Docs and Drive
	scopes := []string{
		"https://www.googleapis.com/auth/documents.readonly",
		"https://www.googleapis.com/auth/drive.readonly",
	}

	config, err := google.JWTConfigFromJSON(credentialsJSON, scopes...)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT config: %w", err)
	}

	// Create a single HTTP client with the JWT config
	httpClient := config.Client(ctx)

	// Initialize Docs service
	docsService, err := docs.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create docs service: %w", err)
	}

	// Initialize Drive service
	driveService, err := drive.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create drive service: %w", err)
	}

	return &Client{
		Docs:  docsService,
		Drive: driveService,
	}, nil
}
