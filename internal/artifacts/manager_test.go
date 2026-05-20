package artifacts_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"bauer/internal/artifacts"
)

func TestNewRunID_Format(t *testing.T) {
	id := artifacts.NewRunID()
	// Format: YYYY-MM-DDTHH-MM-SSZ-{8hex}
	re := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}-\d{2}-\d{2}Z-[0-9a-f]{8}$`)
	if !re.MatchString(id) {
		t.Errorf("NewRunID() = %q, does not match expected format YYYY-MM-DDTHH-MM-SSZ-{8hex}", id)
	}
}

func TestNewRunID_Unique(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 10; i++ {
		id := artifacts.NewRunID()
		if ids[id] {
			t.Errorf("NewRunID() produced duplicate ID: %q", id)
		}
		ids[id] = true
	}
}

func TestStartRun_DirectoryStructure(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := artifacts.NewManager(tmpDir)

	runID, err := mgr.StartRun(artifacts.RunMetadata{
		DocID: "test-doc-123",
		Mode:  "dry-run",
	})
	if err != nil {
		t.Fatalf("StartRun() error = %v", err)
	}
	if runID == "" {
		t.Fatal("StartRun() returned empty runID")
	}

	// Check expected subdirectories
	for _, subdir := range []string{"extraction", "prompts", "outputs", "logs", "screenshots"} {
		dir := filepath.Join(tmpDir, runID, subdir)
		info, err := os.Stat(dir)
		if err != nil {
			t.Errorf("expected directory %s to exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("expected %s to be a directory", dir)
		}
	}

	// Check metadata.json exists with correct status
	metaPath := filepath.Join(tmpDir, runID, "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("metadata.json not found: %v", err)
	}
	var meta artifacts.RunMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("failed to parse metadata.json: %v", err)
	}
	if meta.Status != "in_progress" {
		t.Errorf("metadata.Status = %q, want %q", meta.Status, "in_progress")
	}
	if meta.DocID != "test-doc-123" {
		t.Errorf("metadata.DocID = %q, want %q", meta.DocID, "test-doc-123")
	}
	if meta.RunID != runID {
		t.Errorf("metadata.RunID = %q, want %q", meta.RunID, runID)
	}
}

func TestCompleteRun_AppendsToJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := artifacts.NewManager(tmpDir)

	// Start two runs
	runID1, err := mgr.StartRun(artifacts.RunMetadata{DocID: "doc-1", Mode: "execute"})
	if err != nil {
		t.Fatalf("StartRun #1 error = %v", err)
	}
	runID2, err := mgr.StartRun(artifacts.RunMetadata{DocID: "doc-2", Mode: "dry-run"})
	if err != nil {
		t.Fatalf("StartRun #2 error = %v", err)
	}

	// Complete both
	if err := mgr.CompleteRun(runID1, "success", 3); err != nil {
		t.Fatalf("CompleteRun #1 error = %v", err)
	}
	if err := mgr.CompleteRun(runID2, "failed", 1); err != nil {
		t.Fatalf("CompleteRun #2 error = %v", err)
	}

	// Read runs.jsonl
	indexPath := filepath.Join(tmpDir, "runs.jsonl")
	data, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("runs.jsonl not found: %v", err)
	}

	lines := splitLines(data)
	if len(lines) != 2 {
		t.Fatalf("runs.jsonl has %d lines, want 2", len(lines))
	}

	var entry1 artifacts.RunIndexEntry
	if err := json.Unmarshal([]byte(lines[0]), &entry1); err != nil {
		t.Fatalf("failed to parse line 1: %v", err)
	}
	if entry1.RunID != runID1 {
		t.Errorf("entry1.RunID = %q, want %q", entry1.RunID, runID1)
	}
	if entry1.Status != "success" {
		t.Errorf("entry1.Status = %q, want %q", entry1.Status, "success")
	}
	if entry1.ChunkCount != 3 {
		t.Errorf("entry1.ChunkCount = %d, want 3", entry1.ChunkCount)
	}

	var entry2 artifacts.RunIndexEntry
	if err := json.Unmarshal([]byte(lines[1]), &entry2); err != nil {
		t.Fatalf("failed to parse line 2: %v", err)
	}
	if entry2.RunID != runID2 {
		t.Errorf("entry2.RunID = %q, want %q", entry2.RunID, runID2)
	}
	if entry2.Status != "failed" {
		t.Errorf("entry2.Status = %q, want %q", entry2.Status, "failed")
	}
}

// splitLines splits byte data into non-empty lines.
func splitLines(data []byte) []string {
	var lines []string
	start := 0
	for i, b := range data {
		if b == '\n' {
			line := string(data[start:i])
			if line != "" {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(data) {
		line := string(data[start:])
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}
