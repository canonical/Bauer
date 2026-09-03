package config

import (
	"testing"
)

// validCredentialsJSON is a minimal service-account key that satisfies
// gdocs.ValidateCredentials, which requires the type, private_key,
// client_email, client_id, project_id, and token_uri fields to be non-empty.
const validCredentialsJSON = `{
	"type": "service_account",
	"project_id": "test-project",
	"private_key": "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----\n",
	"client_email": "test@test-project.iam.gserviceaccount.com",
	"client_id": "1234567890",
	"token_uri": "https://oauth2.googleapis.com/token"
}`

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsJSON: []byte(validCredentialsJSON),
				OutputDir:       "bauer-output",
			},
			wantErr: false,
		},
		{
			name: "Missing DocID",
			config: Config{
				DocID:           "",
				CredentialsJSON: []byte(validCredentialsJSON),
			},
			wantErr: true,
		},
		{
			name: "No credentials (assembled from env at runtime)",
			config: Config{
				DocID: "some-doc-id",
			},
			wantErr: false,
		},
		{
			name: "Invalid credentials JSON",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsJSON: []byte(`{"type":"service_account"}`),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
