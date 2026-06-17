package workflow

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func doRequest(t *testing.T, orch *mockOrchestrator, method, body string) (*httptest.ResponseRecorder, APIResponse) {
	t.Helper()
	req := httptest.NewRequest(method, "/workflow", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	ExecuteWorkflowHandler(orch)(rec, req)

	var resp APIResponse
	// Validation errors use a different shape; only decode into APIResponse when
	// the caller expects a workflow response.
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	return rec, resp
}

func TestExecuteWorkflowHandler_Validation(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{
			name:       "rejects non-POST",
			method:     http.MethodGet,
			body:       "",
			wantStatus: http.StatusMethodNotAllowed,
		},
		{
			name:       "rejects invalid JSON",
			method:     http.MethodPost,
			body:       "{not json",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "requires doc_id",
			method:     http.MethodPost,
			body:       `{"parse_only": true, "credentials": "creds.json"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "requires credentials",
			method:     http.MethodPost,
			body:       `{"parse_only": true, "doc_id": "doc-123"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "requires github_repo when not parse-only",
			method:     http.MethodPost,
			body:       `{"doc_id": "doc-123", "credentials": "creds.json", "github_token": "tok"}`,
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "requires github_token when not parse-only",
			method:     http.MethodPost,
			body:       `{"doc_id": "doc-123", "credentials": "creds.json", "github_repo": "owner/repo"}`,
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// A result is provided in case validation unexpectedly passes; the
			// orchestrator should not be reached for these cases anyway.
			orch := &mockOrchestrator{result: parseResultWith(0)}
			rec, _ := doRequest(t, orch, tt.method, tt.body)
			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestExecuteWorkflowHandler_ParseOnlySuccess(t *testing.T) {
	orch := &mockOrchestrator{result: parseResultWith(3)}

	body, err := json.Marshal(APIRequest{
		DocID:       "doc-123",
		Credentials: "creds.json",
		ParseOnly:   true,
		OutputDir:   t.TempDir(),
	})
	if err != nil {
		t.Fatalf("failed to marshal request: %v", err)
	}

	rec, resp := doRequest(t, orch, http.MethodPost, string(body))

	if rec.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d (body: %s)", rec.Code, http.StatusCreated, rec.Body.String())
	}
	if resp.Status != "success" {
		t.Errorf("response status = %q, want %q", resp.Status, "success")
	}
	if resp.Workflow == nil || resp.Workflow.BauerResult.TotalSuggestions != 3 {
		t.Errorf("expected workflow with 3 suggestions, got %+v", resp.Workflow)
	}
}
