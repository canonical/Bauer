package gdocs

import (
	"encoding/json"
	"fmt"
	"os"

	"bauer/internal/env"
)

// Hard-coded, non-sensitive service account fields. Only the sensitive values
// (project_id, private_key, client_email, client_id) are supplied via the
// environment.
const (
	credentialsType     = "service_account"
	credentialsTokenURI = "https://oauth2.googleapis.com/token"
)

// ServiceAccountCredentials represents the structure of a Google service account JSON key file.
type ServiceAccountCredentials struct {
	Type        string `json:"type"`
	ProjectID   string `json:"project_id"`
	PrivateKey  string `json:"private_key"`
	ClientEmail string `json:"client_email"`
	ClientID    string `json:"client_id"`
	AuthURI     string `json:"auth_uri"`
	TokenURI    string `json:"token_uri"`
}

// CredentialsFromEnv assembles the service account credentials JSON from
// individual environment variables. The sensitive values (project_id,
// private_key, client_email, client_id) are read from the environment, while
// the non-sensitive type and token_uri fields are hard-coded.
func CredentialsFromEnv() ([]byte, error) {
	creds := ServiceAccountCredentials{
		Type:        credentialsType,
		TokenURI:    credentialsTokenURI,
		ProjectID:   env.GetGoEnv("PROJECT_ID"),
		PrivateKey:  env.GetGoEnv("PRIVATE_KEY"),
		ClientEmail: env.GetGoEnv("CLIENT_EMAIL"),
		ClientID:    env.GetGoEnv("CLIENT_ID"),
	}

	data, err := json.Marshal(creds)
	if err != nil {
		return nil, fmt.Errorf("failed to build credentials from environment: %w", err)
	}

	if err := ValidateCredentials(data); err != nil {
		return nil, err
	}

	return data, nil
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

	return ValidateCredentials(data)
}

// ValidateCredentials checks that the raw service account JSON is parseable and
// contains the required fields.
func ValidateCredentials(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("credentials are empty")
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

	if creds.ClientID == "" {
		return fmt.Errorf("missing required field: client_id")
	}

	if creds.ProjectID == "" {
		return fmt.Errorf("missing required field: project_id")
	}

	if creds.TokenURI == "" {
		return fmt.Errorf("missing required field: token_uri")
	}

	return nil
}
