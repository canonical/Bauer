package v1

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bauer/cmd/app/types"
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

func newRouteConfig() types.RouteConfig {
	return types.RouteConfig{
		APIConfig: types.APIConfig{
			CredentialsPath: "creds.json",
			BaseOutputDir:   "bauer-output",
		},
		Orchestrator: &mockOrchestrator{},
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
