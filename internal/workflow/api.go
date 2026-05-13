package workflow

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"bauer/internal/orchestrator"
)

// APIRequest represents the API request for executing a workflow
type APIRequest struct {
	// GitHub configuration
	GitHubRepo   string `json:"github_repo" binding:"required"`  // "owner/repo" or HTTPS URL
	GitHubToken  string `json:"github_token" binding:"required"` // Personal access token
	BranchPrefix string `json:"branch_prefix" default:"bauer"`   // Branch naming prefix

	// Bauer configuration
	DocID       string `json:"doc_id" binding:"required"`         // Google Doc ID
	Credentials string `json:"credentials" binding:"required"`    // Path to service account JSON
	ChunkSize   int    `json:"chunk_size" default:"1"`            // Number of chunks
	PageRefresh bool   `json:"page_refresh" default:"false"`      // Page refresh mode
	OutputDir   string `json:"output_dir" default:"bauer-output"` // Output directory
	Model       string `json:"model" default:"gpt-5-mini-high"`   // Copilot model
	DryRun      bool   `json:"dry_run" default:"false"`           // Dry run mode

	// Local repository path
	LocalRepoPath string `json:"local_repo_path" default:"/tmp"` // Where to clone (optional)
}

// APIResponse represents the API response from workflow execution
type APIResponse struct {
	Status    string          `json:"status"` // "success", "partial", "failed"
	Message   string          `json:"message"`
	Workflow  *WorkflowOutput `json:"workflow"`
	Error     string          `json:"error,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// ExecuteWorkflowHandler is an HTTP handler for executing the complete workflow
func ExecuteWorkflowHandler(orch orchestrator.Orchestrator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default()

		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}

		// Parse request
		var req APIRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Error("failed to parse request", "error", err)
			writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
			return
		}

		// Validate request
		if req.GitHubRepo == "" {
			writeError(w, http.StatusBadRequest, "github_repo is required")
			return
		}
		if req.GitHubToken == "" {
			writeError(w, http.StatusBadRequest, "github_token is required")
			return
		}
		if req.DocID == "" {
			writeError(w, http.StatusBadRequest, "doc_id is required")
			return
		}
		if req.Credentials == "" {
			writeError(w, http.StatusBadRequest, "credentials is required")
			return
		}

		// Set defaults
		if req.BranchPrefix == "" {
			req.BranchPrefix = "bauer"
		}
		if req.LocalRepoPath == "" {
			req.LocalRepoPath = "/tmp"
		}
		if req.OutputDir == "" {
			req.OutputDir = "bauer-output"
		}
		if req.Model == "" {
			req.Model = "gpt-5-mini-high"
		}
		if req.ChunkSize == 0 {
			req.ChunkSize = 1
		}

		// Create workflow input
		input := WorkflowInput{
			GitHubRepo:    req.GitHubRepo,
			GitHubToken:   req.GitHubToken,
			BranchPrefix:  req.BranchPrefix,
			DocID:         req.DocID,
			Credentials:   req.Credentials,
			ChunkSize:     req.ChunkSize,
			PageRefresh:   req.PageRefresh,
			OutputDir:     req.OutputDir,
			ArtifactsDir:  "bauer-artifacts",
			Model:         req.Model,
			DryRun:        req.DryRun,
			LocalRepoPath: fmt.Sprintf("%s/%s-%d", req.LocalRepoPath, "bauer-workflow", time.Now().Unix()),
		}

		logger.Info("workflow API request",
			"github_repo", req.GitHubRepo,
			"doc_id", req.DocID,
			"dry_run", req.DryRun,
		)

		// Execute workflow
		ctx := r.Context()
		// Execute workflow — nil agent factory since the workflow handler doesn't
		// currently support Copilot execution (it only does extraction + PR)
		workflowOutput, err := ExecuteWorkflow(ctx, input, orch, nil)

		// Build response
		response := APIResponse{
			Timestamp: time.Now(),
		}

		if workflowOutput != nil {
			response.Status = workflowOutput.Status
			response.Workflow = workflowOutput

			switch workflowOutput.Status {
			case "success":
				response.Message = fmt.Sprintf(
					"Workflow completed successfully. PR: %s",
					workflowOutput.FinalizationInfo.PullRequest.URL,
				)
			case "partial":
				response.Message = fmt.Sprintf(
					"Workflow completed with errors. Branch: %s. Errors: %d",
					workflowOutput.RepositoryInfo.BranchName,
					len(workflowOutput.Errors),
				)
			default:
				response.Message = "Workflow failed"
				if len(workflowOutput.Errors) > 0 {
					response.Error = workflowOutput.Errors[0]
				}
			}
		}

		if err != nil {
			response.Status = "failed"
			response.Message = "Workflow execution error"
			response.Error = err.Error()
			logger.Error("workflow execution error", "error", err)
		}

		// Determine HTTP status code
		statusCode := http.StatusOK
		switch response.Status {
		case "failed":
			statusCode = http.StatusInternalServerError
		case "partial":
			statusCode = http.StatusAccepted
		case "success":
			statusCode = http.StatusCreated
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)

		logger.Info("workflow API response",
			"status", response.Status,
			"http_status", statusCode,
			"duration", workflowOutput.TotalDuration,
		)
	}
}

// Helper functions

func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "error",
		"error":     message,
		"timestamp": time.Now(),
	})
}
