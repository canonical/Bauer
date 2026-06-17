package gdocs

import (
	"encoding/json"
	"fmt"
	"os"
)

// ServiceAccountCredentials represents the structure of a Google service account JSON key file.
type ServiceAccountCredentials struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	AuthURI     string `json:"auth_uri"`
	TokenURI    string `json:"token_uri"`
}

// ValidateCredentialsFile checks if the credentials file exists, is readable, and contains required fields.
func ValidateCredentialsFile(path string) error {
	// Read the credentials file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("credentials file is empty: %s", path)
	}

	// Parse JSON
	var creds ServiceAccountCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return fmt.Errorf("failed to parse credentials JSON: %w", err)
	}

	// Validate required fields
	if creds.Type == "" {
		return fmt.Errorf("missing required field: type")
	}

	if creds.PrivateKey == "" {
		return fmt.Errorf("missing required field: private_key")
	}

	if creds.ClientEmail == "" {
		return fmt.Errorf("missing required field: client_email")
	}

	if creds.ProjectID == "" {
		return fmt.Errorf("missing required field: project_id")
	}

	if creds.TokenURI == "" {
		return fmt.Errorf("missing required field: token_uri")
	}

	return nil
}
