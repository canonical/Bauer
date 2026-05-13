package v1

import (
	"bauer/cmd/app/models/v1"
	"bauer/cmd/app/types"
	"bauer/internal/artifacts"
	"bauer/internal/config"
	"bauer/internal/orchestrator"
	"bauer/internal/source"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// NewAgentFunc creates an orchestrator.Agent. The API server calls this per-request
// so each job gets its own Copilot client pointing at the correct workspace.
type NewAgentFunc func(cwd string) (orchestrator.Agent, error)

func JobPost(rc types.RouteConfig, newAgent NewAgentFunc) func(w http.ResponseWriter, r *http.Request) {
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
			PageRefresh:     payload.PageRefresh,
			CredentialsPath: rc.APIConfig.CredentialsPath,
			OutputDir:       fmt.Sprintf("%s/%s", rc.APIConfig.BaseOutputDir, requestID),
			Model:           rc.APIConfig.Model,
			SummaryModel:    rc.APIConfig.SummaryModel,
		}

		go executeJob(requestID, cfg, rc, newAgent)

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

// executeJob creates a fresh per-request orchestrator (with its own Copilot client)
// so concurrent HTTP handlers never share mutable agent state.
func executeJob(requestID string, cfg config.Config, rc types.RouteConfig, newAgent NewAgentFunc) {
	ctx := context.Background()
	ctx = context.WithValue(ctx, "requestID", requestID)

	// Create a per-request Copilot client so each job targets the right workspace
	agent, err := newAgent(rc.APIConfig.TargetRepo)
	if err != nil {
		slog.Error("failed to create agent", "error", err.Error(), "requestID", requestID)
		return
	}

	// Build a per-request orchestrator to avoid shared mutable state across goroutines
	gdocsAdapter := source.NewGDocsAdapter()
	sources := source.NewManager(gdocsAdapter)
	artMgr := artifacts.NewManager(rc.APIConfig.ArtifactsDir)

	orch := orchestrator.NewOrchestrator(agent, sources, artMgr)

	_, err = orch.Execute(ctx, &cfg)
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
