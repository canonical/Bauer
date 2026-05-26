package v1

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"bauer/cmd/app/types"
	"bauer/internal/artifacts"
	"bauer/internal/config"
	"bauer/internal/copilotcli"
	"bauer/internal/jira"
	"bauer/internal/orchestrator"
	"bauer/internal/source"
)

// JiraWebhookHandler handles POST /api/v1/webhooks/jira.
// It validates a shared secret, extracts the Google Doc ID from the Jira issue payload,
// and runs the orchestrator (extraction + generation) asynchronously so the response
// is immediate. It does not perform Git operations (branch/PR creation).
func JiraWebhookHandler(apiCfg *types.APIConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. Validate shared secret (constant-time comparison prevents timing attacks).
		expectedSecret := os.Getenv("BAUER_JIRA_WEBHOOK_SECRET")
		if expectedSecret != "" {
			if !hmac.Equal([]byte(r.Header.Get("X-Webhook-Secret")), []byte(expectedSecret)) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// 2. Parse payload.
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1MB limit
		var payload jira.WebhookPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		// 3. Extract doc ID from configured custom field.
		fieldKey := firstNonEmpty(os.Getenv("BAUER_JIRA_DOC_FIELD"), "customfield_10100")
		docID := jira.ExtractDocID(&payload, fieldKey)
		if docID == "" {
			slog.Warn("Jira webhook received but no doc ID found",
				slog.String("issue_key", payload.Issue.Key),
				slog.String("field_key", fieldKey),
			)
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// 4. Fire workflow in background so we respond fast.
		go func() {
			tmpDir, err := os.MkdirTemp("", "bauer-jira-*")
			if err != nil {
				slog.Error("failed to create temp dir", slog.String("error", err.Error()))
				return
			}

			cfg := &config.Config{
				DocID:           docID,
				CredentialsPath: firstNonEmpty(os.Getenv("BAUER_CREDENTIALS_PATH"), os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")),
				Model:           firstNonEmpty(os.Getenv("BAUER_MODEL"), apiCfg.Model, "gpt-5-mini-high"),
				ChunkSize:       firstNonZero(1),
				DryRun:          config.BoolPtr(false),
				OutputDir:       tmpDir,
			}
			cfg.ApplyDefaults()

			sources := source.NewManager(cfg.CredentialsPath)
			arts := artifacts.NewManager(firstNonEmpty(os.Getenv("BAUER_ARTIFACTS_DIR"), "./bauer-artifacts"))
			agent, err := copilotcli.NewClient(tmpDir)
			if err != nil {
				slog.Error("failed to create copilot client", slog.String("error", err.Error()))
				return
			}
			orch := orchestrator.New(agent, sources, arts)
			if _, err := orch.Execute(context.Background(), cfg); err != nil {
				slog.Error("Jira webhook workflow failed",
					slog.String("issue_key", payload.Issue.Key),
					slog.String("error", err.Error()),
				)
			}
		}()

		w.WriteHeader(http.StatusAccepted)
	}
}
