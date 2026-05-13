package source

import (
	"context"
	"fmt"
	"testing"

	"bauer/internal/gdocs"
)

// mockAdapter implements Adapter for testing the Manager.
type mockAdapter struct {
	name   string
	result any
	err    error
}

func (m *mockAdapter) Name() string { return m.name }
func (m *mockAdapter) Fetch(_ context.Context, _ Request) (any, error) {
	return m.result, m.err
}

func TestManager_Fetch_WithGDocsResult(t *testing.T) {
	t.Parallel()

	expected := &gdocs.ProcessingResult{
		DocumentTitle: "Test Doc",
		DocumentID:    "doc-123",
	}

	mgr := NewManager(&mockAdapter{name: "gdocs", result: expected})
	bundle, err := mgr.Fetch(context.Background(), Request{DocID: "doc-123", CredentialsPath: "creds.json"})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}

	if bundle.Document == nil {
		t.Fatal("bundle.Document is nil")
	}
	if bundle.Document.DocumentTitle != "Test Doc" {
		t.Fatalf("DocumentTitle = %q, want %q", bundle.Document.DocumentTitle, "Test Doc")
	}
	if bundle.Design != nil {
		t.Fatal("bundle.Design should be nil when no figma adapter is configured")
	}
}

func TestManager_Fetch_NoAdapters(t *testing.T) {
	t.Parallel()

	mgr := NewManager()
	bundle, err := mgr.Fetch(context.Background(), Request{DocID: "doc-123", CredentialsPath: "creds.json"})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	if bundle.Document != nil {
		t.Fatal("bundle.Document should be nil with no adapters")
	}
}

func TestManager_Fetch_AdapterError(t *testing.T) {
	t.Parallel()

	mgr := NewManager(&mockAdapter{name: "error-source", result: nil, err: fmt.Errorf("source unavailable")})
	_, err := mgr.Fetch(context.Background(), Request{DocID: "doc-123", CredentialsPath: "creds.json"})
	if err == nil {
		t.Fatal("expected error from failing adapter")
	}
}

func TestManager_Fetch_UnknownResultType(t *testing.T) {
	t.Parallel()

	mgr := NewManager(&mockAdapter{name: "unknown", result: "unexpected-string"})
	bundle, err := mgr.Fetch(context.Background(), Request{DocID: "doc-123", CredentialsPath: "creds.json"})
	if err != nil {
		t.Fatalf("Fetch() error = %v", err)
	}
	// Unknown types are logged but do not cause a fatal error
	if bundle.Document != nil {
		t.Fatal("bundle.Document should be nil for unknown result type")
	}
}
