package gdocs

import (
	"context"
	"fmt"
	"os"

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

// NewClient creates a new Google Docs and Drive client using the provided credentials file.
func NewClient(ctx context.Context, credentialsPath string) (*Client, error) {
	// Read service account credentials
	credentials, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read service account file: %w", err)
	}

	// Scopes for both Docs and Drive
	scopes := []string{
		"https://www.googleapis.com/auth/documents.readonly",
		"https://www.googleapis.com/auth/drive.readonly",
	}

	config, err := google.JWTConfigFromJSON(credentials, scopes...)
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
