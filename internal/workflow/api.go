package workflow

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"bauer/internal/github"
	"bauer/internal/orchestrator"
)

// APIRequest represents the API request for executing a workflow
type APIRequest struct {
	// GitHub configuration
	GitHubRepo   string `json:"github_repo"`             // "owner/repo" or HTTPS URL
	BranchPrefix string `json:"branch_prefix,omitempty"` // Branch naming prefix

	// Bauer configuration
	DocID        string `json:"doc_id"`                  // Google Doc ID
	ChunkSize    int    `json:"chunk_size,omitempty"`    // Number of chunks
	PageRefresh  bool   `json:"page_refresh,omitempty"`  // Page refresh mode
	Model        string `json:"model,omitempty"`         // Copilot model
	SummaryModel string `json:"summary_model,omitempty"` // Copilot summary model
	DryRun       bool   `json:"dry_run,omitempty"`       // Dry run mode
	FigmaURL     string `json:"figma_url,omitempty"`     // Optional Figma file URL
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
		if req.DocID == "" {
			writeError(w, http.StatusBadRequest, "doc_id is required")
			return
		}

		// Resolve secrets from environment
		token, err := github.GetGitHubToken()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "no GitHub token configured: "+err.Error())
			return
		}
		credentials := firstNonEmpty(os.Getenv("BAUER_CREDENTIALS_PATH"), os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"))
		if credentials == "" {
			writeError(w, http.StatusInternalServerError, "no credentials configured: set BAUER_CREDENTIALS_PATH")
			return
		}

		// Create workflow input (request fields override env defaults)
		input := WorkflowInput{
			GitHubRepo:    req.GitHubRepo,
			GitHubToken:   token,
			BranchPrefix:  firstNonEmpty(req.BranchPrefix, os.Getenv("BAUER_BRANCH_PREFIX"), "bauer"),
			DocID:         req.DocID,
			Credentials:   credentials,
			ChunkSize:     firstNonZero(req.ChunkSize, 1),
			PageRefresh:   req.PageRefresh,
			OutputDir:     firstNonEmpty(os.Getenv("BAUER_OUTPUT_DIR"), "bauer-output"),
			Model:         firstNonEmpty(req.Model, os.Getenv("BAUER_MODEL"), "gpt-5-mini-high"),
			DryRun:        req.DryRun,
			FigmaURL:      req.FigmaURL,
			FigmaToken:    firstNonEmpty(os.Getenv("BAUER_FIGMA_TOKEN"), os.Getenv("FIGMA_TOKEN")),
			LocalRepoPath: fmt.Sprintf("%s/%s-%d", "/tmp", "bauer-workflow", time.Now().Unix()),
		}

		logger.Info("workflow API request",
			"github_repo", req.GitHubRepo,
			"doc_id", req.DocID,
			"dry_run", req.DryRun,
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

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func firstNonZero(vals ...int) int {
	for _, v := range vals {
		if v != 0 {
			return v
		}
	}
	return 0
}
