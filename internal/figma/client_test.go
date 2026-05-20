package figma_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bauer/internal/figma"
)

func TestGetMeta_Success(t *testing.T) {
	meta := figma.FileMeta{
		Name:         "My Design",
		LastModified: "2024-01-15T10:00:00Z",
		Version:      "42",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(meta)
	}))
	defer srv.Close()

	// Override baseURL by using a client pointed at the test server via transport.
	transport := &prefixTransport{base: srv.URL}
	client := figma.NewClientWithHTTP("test-token", &http.Client{Transport: transport})

	got, err := client.GetMeta(context.Background(), "fileKey123")
	if err != nil {
		t.Fatalf("GetMeta error: %v", err)
	}
	if got.Name != meta.Name {
		t.Errorf("Name = %q, want %q", got.Name, meta.Name)
	}
	if got.Version != meta.Version {
		t.Errorf("Version = %q, want %q", got.Version, meta.Version)
	}
}

func TestGetMeta_Unauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	transport := &prefixTransport{base: srv.URL}
	client := figma.NewClientWithHTTP("bad-token", &http.Client{Transport: transport})

	_, err := client.GetMeta(context.Background(), "fileKey123")
	if err == nil {
		t.Fatal("expected error for 401, got nil")
	}
}

func TestGetMeta_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	transport := &prefixTransport{base: srv.URL}
	client := figma.NewClientWithHTTP("test-token", &http.Client{Transport: transport})

	_, err := client.GetMeta(context.Background(), "missingKey")
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
}

func TestGetMeta_RateLimited(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	transport := &prefixTransport{base: srv.URL}
	client := figma.NewClientWithHTTP("test-token", &http.Client{Transport: transport})

	_, err := client.GetMeta(context.Background(), "fileKey")
	if err == nil {
		t.Fatal("expected error for 429, got nil")
	}
}

func TestGetComments_Success(t *testing.T) {
	resolved := "2024-01-16T12:00:00Z"
	resp := figma.CommentsResponse{
		Comments: []figma.Comment{
			{
				ID:         "c1",
				Message:    "Look at this node",
				ClientMeta: figma.CommentClientMeta{NodeID: "1:42"},
				CreatedAt:  "2024-01-15T10:00:00Z",
				User:       figma.CommentUser{Handle: "alice"},
			},
			{
				ID:         "c2",
				Message:    "Resolved comment",
				CreatedAt:  "2024-01-15T11:00:00Z",
				User:       figma.CommentUser{Handle: "bob"},
				ResolvedAt: &resolved,
			},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	transport := &prefixTransport{base: srv.URL}
	client := figma.NewClientWithHTTP("test-token", &http.Client{Transport: transport})

	got, err := client.GetComments(context.Background(), "fileKey123")
	if err != nil {
		t.Fatalf("GetComments error: %v", err)
	}
	if len(got.Comments) != 2 {
		t.Errorf("got %d comments, want 2", len(got.Comments))
	}
}

func TestGetNodes_EmptyIDs(t *testing.T) {
	// When nodeIDs is empty, should return empty response without making an HTTP call.
	client := figma.NewClientWithHTTP("test-token", &http.Client{})
	got, err := client.GetNodes(context.Background(), "fileKey", []string{})
	if err != nil {
		t.Fatalf("GetNodes error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil response")
	}
}

func TestGetImages_EmptyIDs(t *testing.T) {
	client := figma.NewClientWithHTTP("test-token", &http.Client{})
	got, err := client.GetImages(context.Background(), "fileKey", []string{})
	if err != nil {
		t.Fatalf("GetImages error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

// prefixTransport rewrites requests so they go to the test server instead of api.figma.com.
type prefixTransport struct {
	base string
}

func (t *prefixTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Rewrite the host to point to the test server.
	req2 := req.Clone(req.Context())
	req2.URL.Scheme = "http"
	req2.URL.Host = t.base[len("http://"):]
	return http.DefaultTransport.RoundTrip(req2)
}
