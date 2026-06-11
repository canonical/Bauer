package gdocs

import (
	"context"
	"encoding/json"
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

	credentials, err = normalizeCredentials(credentials)
	if err != nil {
		return nil, err
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

// normalizeCredentials translates credentials that use the legacy "google_private_key" field to the expected "private_key" field,
// while ensuring all required fields are present
func normalizeCredentials(data []byte) ([]byte, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse credentials JSON: %w", err)
	}

	if _, ok := raw["private_key"]; !ok {
		if legacyPrivateKey, ok := raw["google_private_key"]; ok {
			raw["private_key"] = legacyPrivateKey
		}
	}

	delete(raw, "google_private_key")

	normalized, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize credentials JSON: %w", err)
	}

	return normalized, nil
}
