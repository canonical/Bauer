package source_test

import (
	"context"
	"testing"

	"bauer/internal/source"
)

func TestNewManager(t *testing.T) {
	mgr := source.NewManager("credentials.json")
	if mgr == nil {
		t.Fatal("NewManager() returned nil")
	}
}

func TestManager_Fetch_EmptyDocID(t *testing.T) {
	mgr := source.NewManager("credentials.json")
	// When DocID is empty, no fetch is performed and an empty bundle is returned.
	bundle, err := mgr.Fetch(context.Background(), source.Request{})
	if err != nil {
		t.Fatalf("Fetch() with empty request error = %v, want nil", err)
	}
	if bundle == nil {
		t.Fatal("Fetch() returned nil bundle")
	}
	if bundle.Document != nil {
		t.Fatal("Fetch() expected nil Document when DocID is empty")
	}
	if bundle.Design != nil {
		t.Fatal("Fetch() expected nil Design when FigmaURL is empty")
	}
}
