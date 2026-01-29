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
				ChunkSize:       1,
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
				ChunkSize:       1,
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
				ChunkSize:       1,
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
				ChunkSize:       1,
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
				ChunkSize:       1,
				Model:           "gpt-4",
				SummaryModel:    "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "Invalid chunk size (negative)",
			config: Config{
				DocID:           "some-doc-id",
				CredentialsPath: validCredsFile,
				ChunkSize:       -1,
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
				ChunkSize:       1,
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
				ChunkSize:       1,
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

func TestChunkSizeDefaults(t *testing.T) {
	// Create a temporary file to act as a valid credentials file
	tmpDir := t.TempDir()
	validCredsFile := filepath.Join(tmpDir, "creds.json")
	if err := os.WriteFile(validCredsFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create temp creds file: %v", err)
	}

	tests := []struct {
		name              string
		chunkSizeFlag     int
		pageRefreshFlag   bool
		expectedChunkSize int
	}{
		{
			name:              "Default chunk size (no flags)",
			chunkSizeFlag:     0,
			pageRefreshFlag:   false,
			expectedChunkSize: 1,
		},
		{
			name:              "Page refresh flag sets chunk size to 5",
			chunkSizeFlag:     0,
			pageRefreshFlag:   true,
			expectedChunkSize: 5,
		},
		{
			name:              "Explicit chunk size overrides default",
			chunkSizeFlag:     10,
			pageRefreshFlag:   false,
			expectedChunkSize: 10,
		},
		{
			name:              "Explicit chunk size overrides page refresh default",
			chunkSizeFlag:     3,
			pageRefreshFlag:   true,
			expectedChunkSize: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic from Load()
			effectiveChunkSize := tt.chunkSizeFlag
			if effectiveChunkSize == 0 {
				if tt.pageRefreshFlag {
					effectiveChunkSize = 5
				} else {
					effectiveChunkSize = 1
				}
			}

			if effectiveChunkSize != tt.expectedChunkSize {
				t.Errorf("Expected chunk size %d, got %d", tt.expectedChunkSize, effectiveChunkSize)
			}

			// Create config with computed chunk size
			cfg := Config{
				DocID:           "test-doc-id",
				CredentialsPath: validCredsFile,
				ChunkSize:       effectiveChunkSize,
				PageRefresh:     tt.pageRefreshFlag,
				OutputDir:       "bauer-output",
				Model:           "gpt-5-mini-high",
				SummaryModel:    "gpt-5-mini-high",
			}

			// Validate should pass
			if err := cfg.Validate(); err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}

			// Verify the chunk size is what we expect
			if cfg.ChunkSize != tt.expectedChunkSize {
				t.Errorf("Config chunk size = %d, expected %d", cfg.ChunkSize, tt.expectedChunkSize)
			}
		})
	}
}
