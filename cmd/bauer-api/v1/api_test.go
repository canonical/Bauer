package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bauer/cmd/bauer-api/types"
	"bauer/internal/config"
	"bauer/internal/orchestrator"
)

// mockOrchestrator is a test double for orchestrator.Orchestrator. It lets the
// handlers be exercised without touching Google Docs or the network.
type mockOrchestrator struct {
	result *orchestrator.OrchestrationResult
	err    error
}

func (m *mockOrchestrator) Execute(_ context.Context, _ *config.Config) (*orchestrator.OrchestrationResult, error) {
	return m.result, m.err
}

func newRouteConfig(orch orchestrator.Orchestrator, outputDir string) types.RouteConfig {
	return types.RouteConfig{
		APIConfig: types.APIConfig{
			CredentialsPath: "creds.json",
			BaseOutputDir:   outputDir,
		},
		Orchestrator: orch,
	}
}

// withRequestID attaches a request ID to the request context, mirroring what the
// RequestTrace middleware does in production.
func withRequestID(r *http.Request, id string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), "requestID", id))
}

// TestGetHealth covers GET /api/v1, which reports API liveness.
func TestGetHealth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1", nil)
	rec := httptest.NewRecorder()

	GetHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp types.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if resp.Code != http.StatusOK {
		t.Errorf("response code = %d, want %d", resp.Code, http.StatusOK)
	}
	if resp.Error != "" {
		t.Errorf("response error = %q, want empty", resp.Error)
	}
}

// TestPost covers POST /api/v1, which triggers the orchestration process.
func TestPost(t *testing.T) {
	payload := struct {
		GithubRepo string `json:"github_repo"`
		DocId      string `json:"doc_id"`
		ParseOnly  bool   `json:"parse_only"`
	}{
		GithubRepo: "canonical/canonical.com",
		DocId:      "1WJ-N_Xkkx4r_6knxW7h200oIDyi4mVMzgh1xYt5xaU0",
		ParseOnly:  true,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/v1", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	// The mock returns a populated result so ExecuteWorkflow can write the
	// parse-result file; output goes to a temp dir to avoid source-tree artifacts.
	mock := &mockOrchestrator{
		result: &orchestrator.OrchestrationResult{
			ParseResult: &orchestrator.ParseResult{},
		},
	}
	rc := newRouteConfig(mock, t.TempDir())

	WorkflowPost(rc)(rec, withRequestID(req, "test-request-id"))

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}

	var resp types.Response
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}
	if resp.Code != http.StatusCreated {
		t.Errorf("response code = %d, want %d", resp.Code, http.StatusCreated)
	}
	if resp.Error != "" {
		t.Errorf("response error = %q, want empty", resp.Error)
	}
}
