package config

import "testing"

func setValidGoogleEnv(t *testing.T) {
	t.Helper()

	t.Setenv("GOOGLE_TYPE", "service_account")
	t.Setenv("GOOGLE_PROJECT_ID", "test-project")
	t.Setenv("GOOGLE_PRIVATE_KEY_ID", "test-key-id")
	t.Setenv("GOOGLE_PRIVATE_KEY", "-----BEGIN PRIVATE KEY-----\\nabc\\n-----END PRIVATE KEY-----\\n")
	t.Setenv("GOOGLE_CLIENT_EMAIL", "test@example.iam.gserviceaccount.com")
	t.Setenv("GOOGLE_CLIENT_ID", "1234567890")
	t.Setenv("GOOGLE_AUTH_URI", "https://accounts.google.com/o/oauth2/auth")
	t.Setenv("GOOGLE_TOKEN_URI", "https://oauth2.googleapis.com/token")
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		setup   func(t *testing.T)
		wantErr bool
	}{
		{
			name: "Valid config",
			config: Config{
				DocID:        "some-doc-id",
				ChunkSize:    1,
				OutputDir:    "bauer-output",
				Model:        "gpt-4",
				SummaryModel: "gpt-4",
			},
			setup:   setValidGoogleEnv,
			wantErr: false,
		},
		{
			name: "Missing DocID",
			config: Config{
				DocID:        "",
				ChunkSize:    1,
				Model:        "gpt-4",
				SummaryModel: "gpt-4",
			},
			setup:   setValidGoogleEnv,
			wantErr: true,
		},
		{
			name: "Missing required env",
			config: Config{
				DocID:        "some-doc-id",
				ChunkSize:    1,
				Model:        "gpt-4",
				SummaryModel: "gpt-4",
			},
			setup: func(t *testing.T) {
				setValidGoogleEnv(t)
				t.Setenv("GOOGLE_CLIENT_EMAIL", "")
			},
			wantErr: true,
		},
		{
			name: "Invalid chunk size (negative)",
			config: Config{
				DocID:        "some-doc-id",
				ChunkSize:    -1,
				Model:        "gpt-4",
				SummaryModel: "gpt-4",
			},
			setup:   setValidGoogleEnv,
			wantErr: true,
		},
		{
			name: "Valid config with default model",
			config: Config{
				DocID:        "some-doc-id",
				ChunkSize:    1,
				OutputDir:    "bauer-output",
				Model:        "gpt-5-mini-high",
				SummaryModel: "gpt-5-mini-high",
			},
			setup:   setValidGoogleEnv,
			wantErr: false,
		},
		{
			name: "Valid config with empty model (should be allowed, has default)",
			config: Config{
				DocID:        "some-doc-id",
				ChunkSize:    1,
				OutputDir:    "bauer-output",
				Model:        "",
				SummaryModel: "",
			},
			setup:   setValidGoogleEnv,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChunkSizeDefaults(t *testing.T) {
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
			setValidGoogleEnv(t)

			cfg := Config{
				DocID:        "test-doc-id",
				ChunkSize:    tt.chunkSizeFlag,
				PageRefresh:  tt.pageRefreshFlag,
				OutputDir:    "bauer-output",
				Model:        "gpt-5-mini-high",
				SummaryModel: "gpt-5-mini-high",
			}

			if err := cfg.Validate(); err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}

			if cfg.ChunkSize != tt.expectedChunkSize {
				t.Errorf("Config chunk size = %d, expected %d", cfg.ChunkSize, tt.expectedChunkSize)
			}
		})
	}
}
