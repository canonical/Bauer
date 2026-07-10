package config

import (
	"os"
	"path/filepath"
	"testing"
)

// validCredentialsJSON is a minimal service-account key that satisfies
// gdocs.ValidateCredentialsFile, which requires the type, private_key,
// client_email, project_id, and token_uri fields to be non-empty.
const validCredentialsJSON = `{
	"type": "service_account",
	"project_id": "test-project",
	"private_key": "-----BEGIN PRIVATE KEY-----\nFAKE\n-----END PRIVATE KEY-----\n",
	"client_email": "test@test-project.iam.gserviceaccount.com",
	"token_uri": "https://oauth2.googleapis.com/token"
}`

// writeValidCreds writes a valid service-account credentials file into dir and
// returns its path, failing the test if the file cannot be created.
func writeValidCreds(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "creds.json")
	if err := os.WriteFile(path, []byte(validCredentialsJSON), 0644); err != nil {
		t.Fatalf("Failed to create temp creds file: %v", err)
	}
	return path
}

func TestConfig_Validate(t *testing.T) {
	tmpDir := t.TempDir()
	validCredsFile := writeValidCreds(t, tmpDir)

	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: validCredsFile,
				OutputDir:       "bauer-output",
			},
			wantErr: false,
		},
		{
			name: "Missing DocID",
			config: Config{
				DocID:           "",
				CredentialsPath: validCredsFile,
			},
			wantErr: true,
		},
		{
			name: "Missing CredentialsPath",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: "",
			},
			wantErr: true,
		},
		{
			name: "Credentials file does not exist",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: filepath.Join(tmpDir, "non-existent.json"),
			},
			wantErr: true,
		},
		{
			name: "Credentials path is a directory",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: tmpDir,
			},
			wantErr: true,
		},
		{
			name: "Valid config with default model",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: validCredsFile,
				OutputDir:       "bauer-output",
			},
			wantErr: false,
		},
		{
			name: "Valid config with empty model (should be allowed, has default)",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: validCredsFile,
				OutputDir:       "bauer-output",
			},
			wantErr: false,
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
