package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_Validate(t *testing.T) {
	// Create a temporary file to act as a valid credentials file
	tmpDir := t.TempDir()
	validCredsFile := filepath.Join(tmpDir, "creds.json")
	if err := os.WriteFile(validCredsFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create temp creds file: %v", err)
	}

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
				ChunkSize:       10,
				OutputDir:       "bauer-output",
				Model:           "gpt-4",
				SummaryModel:    "gpt-4",
			},
			wantErr: false,
		},
		{
			name: "Missing DocID",
			config: Config{
				DocID:           "",
				CredentialsPath: validCredsFile,
				ChunkSize:       10,
				Model:           "gpt-4",
				SummaryModel:    "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "Missing CredentialsPath",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: "",
				ChunkSize:       10,
				Model:           "gpt-4",
				SummaryModel:    "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "Credentials file does not exist",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: filepath.Join(tmpDir, "non-existent.json"),
				ChunkSize:       10,
				Model:           "gpt-4",
				SummaryModel:    "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "Credentials path is a directory",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: tmpDir,
				ChunkSize:       10,
				Model:           "gpt-4",
				SummaryModel:    "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "Invalid chunk size",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: validCredsFile,
				ChunkSize:       0,
				Model:           "gpt-4",
				SummaryModel:    "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "Valid config with default model",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: validCredsFile,
				ChunkSize:       10,
				OutputDir:       "bauer-output",
				Model:           "gpt-5-mini-high",
				SummaryModel:    "gpt-5-mini-high",
			},
			wantErr: false,
		},
		{
			name: "Valid config with empty model (should be allowed, has default)",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: validCredsFile,
				ChunkSize:       10,
				OutputDir:       "bauer-output",
				Model:           "",
				SummaryModel:    "",
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
