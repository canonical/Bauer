package v1

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"bauer/cmd/app/types"
	"bauer/internal/artifacts"
	"bauer/internal/config"
	"bauer/internal/copilotcli"
	"bauer/internal/github"
	"bauer/internal/orchestrator"
	"bauer/internal/source"
)

// IssueRequest is the request body for POST /api/v1/issues.
type IssueRequest struct {
	DocID       string `json:"doc_id"`
	GitHubRepo  string `json:"github_repo"`
	ChunkSize   int    `json:"chunk_size,omitempty"`
	PageRefresh bool   `json:"page_refresh,omitempty"`
	Model       string `json:"model,omitempty"`
	FigmaURL    string `json:"figma_url,omitempty"`
}

// IssuesHandler runs the orchestrator in dry-run mode, formats a GitHub issue body,
// creates the issue via the GitHub API, and returns the issue URL and number.
func IssuesHandler(apiCfg *types.APIConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req IssueRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			httpError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		if req.DocID == "" || req.GitHubRepo == "" {
			httpError(w, http.StatusBadRequest, "doc_id and github_repo are required")
			return
		}

		token, err := github.GetGitHubToken()
		if err != nil {
			httpError(w, http.StatusInternalServerError, "GitHub token not configured")
			return
		}

		credsPath := firstNonEmpty(
			apiCfg.CredentialsPath,
			os.Getenv("BAUER_CREDENTIALS_PATH"),
			os.Getenv("GOOGLE_APPLICATION_CREDENTIALS"),
		)
		if credsPath == "" {
			httpError(w, http.StatusInternalServerError, "credentials not configured (set BAUER_CREDENTIALS_PATH)")
			return
		}

		if req.ChunkSize < 0 {
			httpError(w, http.StatusBadRequest, "chunk_size must be a positive integer")
			return
		}

		tmpDir, err := os.MkdirTemp("", "bauer-issues-*")
		if err != nil {
			httpError(w, http.StatusInternalServerError, "failed to create temp directory")
			return
		}

		cfg := &config.Config{
			DocID:           req.DocID,
			CredentialsPath: credsPath,
			Model:           firstNonEmpty(req.Model, apiCfg.Model, "gpt-5-mini-high"),
			ChunkSize:       firstNonZero(req.ChunkSize, 1),
			DryRun:          config.BoolPtr(true),
			OutputDir:       tmpDir,
			FigmaURL:        req.FigmaURL,
			FigmaToken:      os.Getenv("BAUER_FIGMA_TOKEN"),
		}
		if req.PageRefresh {
			cfg.PageRefresh = config.BoolPtr(true)
		}
		cfg.ApplyDefaults()

		sources := source.NewManager(cfg.CredentialsPath)
		arts := artifacts.NewManager(firstNonEmpty(os.Getenv("BAUER_ARTIFACTS_DIR"), "./bauer-artifacts"))
		copilotAgent, err := copilotcli.NewClient(tmpDir)
		if err != nil {
			httpError(w, http.StatusInternalServerError, "failed to create copilot client")
			return
		}
		orch := orchestrator.New(copilotAgent, sources, arts)

		result, err := orch.Execute(r.Context(), cfg)
		if err != nil {
			httpError(w, http.StatusInternalServerError, fmt.Sprintf("orchestration failed: %s", err))
			return
		}

		parts := strings.Split(req.GitHubRepo, "/")
		if len(parts) != 2 {
			httpError(w, http.StatusBadRequest, "github_repo must be in owner/repo format")
			return
		}
		repoFull := parts[0] + "/" + parts[1]

		title := fmt.Sprintf("BAU: Apply suggestions from doc %s", req.DocID)
		body := formatIssueBody(result, req.DocID)

		issueURL, issueNum, err := github.CreateIssue(r.Context(), token, repoFull, title, body)
		if err != nil {
			httpError(w, http.StatusInternalServerError, fmt.Sprintf("creating GitHub issue: %s", err))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"status":       "success",
			"issue_url":    issueURL,
			"issue_number": issueNum,
		})
	}
}

func formatIssueBody(result *orchestrator.OrchestrationResult, docID string) string {
	var sb strings.Builder
	sb.WriteString("## BAU Documentation Improvement Plan\n\n")
	sb.WriteString(fmt.Sprintf("**Doc ID**: `%s`\n\n", docID))
	if result.ExtractionBundle != nil && result.ExtractionBundle.Document != nil {
		sb.WriteString(fmt.Sprintf("**Document**: %s\n\n", result.ExtractionBundle.Document.DocumentTitle))
	}
	sb.WriteString(fmt.Sprintf("**Chunks generated**: %d\n\n", len(result.Chunks)))
	sb.WriteString("### Prompt files\n\n")
	for _, c := range result.Chunks {
		sb.WriteString(fmt.Sprintf("- Chunk %d: %d location(s)\n", c.ChunkNumber, c.LocationCount))
	}
	return sb.String()
}
