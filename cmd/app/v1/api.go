package v1

import (
	"bauer/cmd/app/models/v1"
	"bauer/cmd/app/types"
	"bauer/internal/config"
	"bauer/internal/github"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
)

func JobPost(rc types.RouteConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID, ok := r.Context().Value("requestID").(string)
		if !ok || requestID == "" {
			err := types.InternalError(fmt.Errorf("missing request ID")).Render(w, r)
			if err != nil {
				slog.Error("error writing response", "error", err.Error())
			}
			return
		}
		if r.Method != "POST" {
			err := types.NotAllowed(fmt.Errorf("invalid HTTP method: %s", r.Method)).Render(w, r)
			if err != nil {
				slog.Error("error writing response", "error", err.Error(), "requestID", requestID)
			}
			return
		}
		payload, err := getJobFromRequest(w, r, requestID)
		if err != nil {
			return
		}
		cfg := config.Config{
			DocID:           payload.DocID,
			ChunkSize:       payload.ChunkSize,
			PageRefresh:     config.BoolPtr(payload.PageRefresh),
			CredentialsPath: rc.APIConfig.CredentialsPath,
			OutputDir:       fmt.Sprintf("%s/%s", rc.APIConfig.BaseOutputDir, requestID),
			Model:           rc.APIConfig.Model,
			SummaryModel:    rc.APIConfig.SummaryModel,
		}

		go executeJob(requestID, cfg, rc)

		err = types.Accepted().Render(w, r)
		if err != nil {
			slog.Error("error writing response", "error", err.Error(), "requestID", requestID)
		}
	}
}

func getJobFromRequest(w http.ResponseWriter, r *http.Request, requestID string) (*models.JobPost, error) {
	payload := models.JobPost{}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		slog.Error("failed to decode request body", "error", err.Error(), "requestID", requestID)
		err := types.BadRequest(fmt.Errorf("invalid request body: %w", err)).Render(w, r)
		if err != nil {
			slog.Error("error writing response", "error", err.Error(), "requestID", requestID)
		}
		return nil, err
	}
	return &payload, nil
}

func executeJob(requestID string, cfg config.Config, rc types.RouteConfig) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "requestID", requestID)

	_, err := rc.Orchestrator.Execute(ctx, &cfg)
	if err != nil {
		slog.Error("job execution failed",
			"error", err.Error(),
			"requestID", requestID,
		)
		return
	}

	slog.Info("job executed successfully",
		"requestID", requestID,
	)
}


func GetHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := types.Success().Render(w, r)
	if err != nil {
		slog.Error("error writing response", "error", err.Error())
	}
}

// ReadinessHandler checks that all required runtime dependencies are present.
// It must be registered on the public (unauthenticated) mux so that K8s readiness
// probes — which cannot send bearer tokens — can call it.
func ReadinessHandler(apiCfg *types.APIConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
	failures := map[string]string{}

	// Check credentials — prefer config flag, fall back to env vars.
	credsPath := firstNonEmpty(
		apiCfg.CredentialsPath,
		os.Getenv("BAUER_CREDENTIALS_PATH"),
		os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
	)
	if credsPath == "" {
		failures["credentials"] = "not configured (set BAUER_CREDENTIALS_PATH)"
	} else if _, err := os.Stat(credsPath); err != nil {
		slog.Warn("credentials file not readable", slog.String("error", err.Error()))
		failures["credentials"] = "credentials file is not readable (check server logs for details)"
	}

	// Check GitHub token
	if _, err := github.GetGitHubToken(); err != nil {
		failures["github_token"] = "not configured (set BAUER_GITHUB_TOKEN or run 'gh auth login')"
	}

	// Check gh CLI
	if _, err := exec.LookPath("gh"); err != nil {
		failures["gh_cli"] = "not found in PATH"
	}

	w.Header().Set("Content-Type", "application/json")
	if len(failures) > 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]any{
			"status":  "not ready",
			"missing": failures,
		})
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"status": "ready"})
	}
}