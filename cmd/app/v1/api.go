package v1

import (
	"bauer/cmd/app/models/v1"
	"bauer/cmd/app/types"
	"bauer/internal/github"
	"bauer/internal/workflow"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// WorkflowPost handles requests to trigger the PR-creation workflow on a target
// repository. It extracts suggestions from a Google Doc, pushes the parse result
// to a branch, and opens a Copilot-assigned issue that drives PR creation.
//
// Secrets are never read from the request body: the service account credentials
// come from the server configuration and the GitHub token is resolved from the
// process environment / gh CLI.
func WorkflowPost(rc types.RouteConfig) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID, ok := r.Context().Value("requestID").(string)
		if !ok || requestID == "" {
			if err := types.InternalError(fmt.Errorf("missing request ID")).Render(w, r); err != nil {
				slog.Error("error writing response", "error", err.Error())
			}
			return
		}

		payload, err := getWorkflowFromRequest(w, r, requestID)
		if err != nil {
			return
		}

		if payload.DocID == "" {
			if err := types.BadRequest(fmt.Errorf("doc_id is required")).Render(w, r); err != nil {
				slog.Error("error writing response", "error", err.Error(), "requestID", requestID)
			}
			return
		}
		if payload.GitHubRepo == "" {
			if err := types.BadRequest(fmt.Errorf("github_repo is required")).Render(w, r); err != nil {
				slog.Error("error writing response", "error", err.Error(), "requestID", requestID)
			}
			return
		}

		// Resolve the GitHub token server-side; it is never accepted in the body.
		token, err := github.GetGitHubToken()
		if err != nil {
			slog.Error("failed to resolve github token", "error", err.Error(), "requestID", requestID)
			if err := types.InternalError(fmt.Errorf("github authentication not configured")).Render(w, r); err != nil {
				slog.Error("error writing response", "error", err.Error(), "requestID", requestID)
			}
			return
		}

		branchPrefix := payload.BranchPrefix
		if branchPrefix == "" {
			branchPrefix = "bauer"
		}

		input := workflow.WorkflowInput{
			GitHubRepo:    payload.GitHubRepo,
			GitHubToken:   token,
			BranchPrefix:  branchPrefix,
			DocID:         payload.DocID,
			Credentials:   rc.APIConfig.CredentialsPath,
			ChunkSize:     payload.ChunkSize,
			PageRefresh:   payload.PageRefresh,
			OutputDir:     fmt.Sprintf("%s/%s", rc.APIConfig.BaseOutputDir, requestID),
			ParseOnly:     false,
			LocalRepoPath: fmt.Sprintf("/tmp/bauer-workflow-%s", requestID),
		}

		slog.Info("workflow request received",
			"requestID", requestID,
			"github_repo", payload.GitHubRepo,
			"doc_id", payload.DocID,
		)

		out, err := workflow.ExecuteWorkflow(r.Context(), input, rc.Orchestrator)
		if err != nil {
			slog.Error("workflow execution failed", "error", err.Error(), "requestID", requestID)
			resp := &types.WorkflowResponse{
				Code:   http.StatusInternalServerError,
				Status: "failed",
				Error:  err.Error(),
			}
			if renderErr := resp.Render(w, r); renderErr != nil {
				slog.Error("error writing response", "error", renderErr.Error(), "requestID", requestID)
			}
			return
		}

		slog.Info("workflow executed successfully",
			"requestID", requestID,
			"issue_url", out.FinalizationInfo.Issue.URL,
			"branch", out.RepositoryInfo.BranchName,
		)

		resp := &types.WorkflowResponse{
			Code:     http.StatusCreated,
			Status:   out.Status,
			IssueURL: out.FinalizationInfo.Issue.URL,
			Branch:   out.RepositoryInfo.BranchName,
		}
		if err := resp.Render(w, r); err != nil {
			slog.Error("error writing response", "error", err.Error(), "requestID", requestID)
		}
	}
}

func getWorkflowFromRequest(w http.ResponseWriter, r *http.Request, requestID string) (*models.WorkflowPost, error) {
	payload := models.WorkflowPost{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		slog.Error("failed to decode request body", "error", err.Error(), "requestID", requestID)
		if renderErr := types.BadRequest(fmt.Errorf("invalid request body: %w", err)).Render(w, r); renderErr != nil {
			slog.Error("error writing response", "error", renderErr.Error(), "requestID", requestID)
		}
		return nil, err
	}
	return &payload, nil
}

// GetHealth returns the liveness status of the API.
func GetHealth(w http.ResponseWriter, r *http.Request) {
	if err := types.Success().Render(w, r); err != nil {
		slog.Error("error writing response", "error", err.Error())
	}
}
