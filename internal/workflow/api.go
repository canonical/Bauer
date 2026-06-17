package workflow

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"bauer/internal/orchestrator"
)

// APIRequest represents the API request for executing a workflow.
//
// Defaults are applied by the handler (see ExecuteWorkflowHandler), not by
// struct tags: encoding/json does not honor `default:"..."` tags, so the
// defaults live in code to remain a single source of truth.
type APIRequest struct {
	// GitHub configuration
	GitHubRepo   string `json:"github_repo"`   // "owner/repo" or HTTPS URL (optional; required if not parse_only)
	GitHubToken  string `json:"github_token"`  // Personal access token (optional; required if not parse_only)
	BranchPrefix string `json:"branch_prefix"` // Branch naming prefix (default: "bauer")

	// Bauer configuration
	DocID       string `json:"doc_id"`      // Google Doc ID (required)
	Credentials string `json:"credentials"` // Path to service account JSON (required)
	ChunkSize   int    `json:"chunk_size"`  // Number of chunks (default: 1)
	PageRefresh bool   `json:"page_refresh"`
	OutputDir   string `json:"output_dir"` // Output directory (default: "bauer-output")
	ParseOnly   bool   `json:"parse_only"`

	// Local repository path
	LocalRepoPath string `json:"local_repo_path"` // Where to clone (default: "/tmp")
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
		// github_repo and github_token are required only when NOT in parse-only mode
		if !req.ParseOnly && req.GitHubRepo == "" {
			writeError(w, http.StatusBadRequest, "github_repo is required (unless parse_only=true)")
			return
		}
		if !req.ParseOnly && req.GitHubToken == "" {
			writeError(w, http.StatusBadRequest, "github_token is required (unless parse_only=true)")
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
			ParseOnly:     req.ParseOnly,
			LocalRepoPath: fmt.Sprintf("%s/%s-%d", req.LocalRepoPath, "bauer-workflow", time.Now().Unix()),
		}

		logger.Info("workflow API request",
			"github_repo", req.GitHubRepo,
			"doc_id", req.DocID,
			"parse_only", req.ParseOnly,
			"mode", map[bool]string{true: "parse-only", false: "parse-and-issue"}[req.ParseOnly],
		)

		// Execute workflow
		ctx := r.Context()
		workflowOutput, err := ExecuteWorkflow(ctx, input, orch)

		// Build response
		response := APIResponse{
			Timestamp: time.Now(),
		}

		if workflowOutput != nil {
			response.Status = workflowOutput.Status
			response.Workflow = workflowOutput

			switch workflowOutput.Status {
			case statusSuccess:
				if workflowOutput.FinalizationInfo.Issue.URL != "" {
					response.Message = fmt.Sprintf(
						"Workflow completed successfully. Issue: %s",
						workflowOutput.FinalizationInfo.Issue.URL,
					)
				} else {
					response.Message = fmt.Sprintf(
						"Workflow completed successfully. Output file: %s",
						workflowOutput.OutputFile,
					)
				}
			case statusPartial:
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
			response.Status = statusFailed
			response.Message = "Workflow execution error"
			response.Error = err.Error()
			logger.Error("workflow execution error", "error", err)
		}

		// Determine HTTP status code
		statusCode := http.StatusOK
		switch response.Status {
		case statusFailed:
			statusCode = http.StatusInternalServerError
		case statusPartial:
			statusCode = http.StatusAccepted
		case statusSuccess:
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
