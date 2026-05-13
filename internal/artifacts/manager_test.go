package artifacts

import (
	"bufio"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewRun_CreatesDirectoryStructure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewManager(dir)

	runID, err := mgr.NewRun(context.Background())
	if err != nil {
		t.Fatalf("NewRun() error = %v", err)
	}

	if runID == "" {
		t.Fatal("runID is empty")
	}

	runDir := filepath.Join(dir, runID)
	subdirs := []string{"extraction", "prompts", "outputs", "logs", "screenshots"}
	for _, sub := range subdirs {
		info, err := os.Stat(filepath.Join(runDir, sub))
		if err != nil {
			t.Fatalf("stat %s: %v", sub, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", sub)
		}
	}
}

func TestAppendRun_AppendOnly(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewManager(dir)

	meta1 := RunMeta{
		RunID:       "run-001",
		StartedAt:   "2026-05-13T10:00:00Z",
		CompletedAt: "2026-05-13T10:01:00Z",
		Status:      "success",
		DocID:       "doc-1",
		Mode:        "execute",
		ChunkCount:  3,
		ArtifactDir: "bauer-artifacts/run-001",
	}
	meta2 := RunMeta{
		RunID:       "run-002",
		StartedAt:   "2026-05-13T11:00:00Z",
		CompletedAt: "2026-05-13T11:02:00Z",
		Status:      "failed",
		DocID:       "doc-2",
		Mode:        "dry-run",
		ChunkCount:  0,
		ArtifactDir: "bauer-artifacts/run-002",
	}

	if err := mgr.AppendRun(meta1); err != nil {
		t.Fatalf("AppendRun() error = %v", err)
	}
	if err := mgr.AppendRun(meta2); err != nil {
		t.Fatalf("AppendRun() error = %v", err)
	}

	// Read runs.jsonl and verify both entries exist
	f, err := os.Open(filepath.Join(dir, "runs.jsonl"))
	if err != nil {
		t.Fatalf("open runs.jsonl: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines in runs.jsonl, got %d", len(lines))
	}

	var parsed1, parsed2 RunMeta
	if err := json.Unmarshal([]byte(lines[0]), &parsed1); err != nil {
		t.Fatalf("parse line 1: %v", err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &parsed2); err != nil {
		t.Fatalf("parse line 2: %v", err)
	}

	if parsed1.RunID != "run-001" || parsed2.RunID != "run-002" {
		t.Fatalf("unexpected run IDs: %q, %q", parsed1.RunID, parsed2.RunID)
	}
}

func TestWriteMetadata(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewManager(dir)

	meta := RunMeta{RunID: "run-001", Status: "success"}
	if err := mgr.WriteMetadata("run-001", meta); err != nil {
		t.Fatalf("WriteMetadata() error = %v", err)
	}

	// Verify the parent dir was created (NewRun wasn't called, so it should auto-create)
	data, err := os.ReadFile(filepath.Join(dir, "run-001", "metadata.json"))
	if err != nil {
		t.Fatalf("read metadata.json: %v", err)
	}

	var parsed RunMeta
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse metadata.json: %v", err)
	}
	if parsed.RunID != "run-001" {
		t.Fatalf("RunID = %q, want %q", parsed.RunID, "run-001")
	}
}

func TestWriteExtraction(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewManager(dir)

	data := map[string]string{"key": "value"}
	if err := mgr.WriteExtraction("run-001", "gdocs.json", data); err != nil {
		t.Fatalf("WriteExtraction() error = %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(dir, "run-001", "extraction", "gdocs.json"))
	if err != nil {
		t.Fatalf("read gdocs.json: %v", err)
	}
	if !strings.Contains(string(raw), "key") {
		t.Fatalf("gdocs.json content doesn't contain 'key': %s", raw)
	}
}

func TestWritePromptAndOutput(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewManager(dir)

	if err := mgr.WritePrompt("run-001", "chunk-1-of-3.md", "prompt content"); err != nil {
		t.Fatalf("WritePrompt() error = %v", err)
	}
	if err := mgr.WriteOutput("run-001", "chunk-1-output.md", "output content"); err != nil {
		t.Fatalf("WriteOutput() error = %v", err)
	}

	promptContent, err := os.ReadFile(filepath.Join(dir, "run-001", "prompts", "chunk-1-of-3.md"))
	if err != nil {
		t.Fatalf("read prompt: %v", err)
	}
	if string(promptContent) != "prompt content" {
		t.Fatalf("prompt = %q, want %q", promptContent, "prompt content")
	}

	outputContent, err := os.ReadFile(filepath.Join(dir, "run-001", "outputs", "chunk-1-output.md"))
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(outputContent) != "output content" {
		t.Fatalf("output = %q, want %q", outputContent, "output content")
	}
}

func TestAppendLog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewManager(dir)

	entry := LogEntry{Timestamp: "2026-05-13T10:00:00Z", Step: "extraction", Message: "started"}
	if err := mgr.AppendLog("run-001", entry); err != nil {
		t.Fatalf("AppendLog() error = %v", err)
	}

	f, err := os.Open(filepath.Join(dir, "run-001", "logs", "execution.jsonl"))
	if err != nil {
		t.Fatalf("open execution.jsonl: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		t.Fatal("execution.jsonl is empty")
	}

	var parsed LogEntry
	if err := json.Unmarshal([]byte(scanner.Text()), &parsed); err != nil {
		t.Fatalf("parse log line: %v", err)
	}
	if parsed.Step != "extraction" {
		t.Fatalf("Step = %q, want %q", parsed.Step, "extraction")
	}
}

func TestEnsureScreenshotsDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mgr := NewManager(dir)

	screenshotsDir, err := mgr.EnsureScreenshotsDir("run-001")
	if err != nil {
		t.Fatalf("EnsureScreenshotsDir() error = %v", err)
	}

	info, err := os.Stat(screenshotsDir)
	if err != nil {
		t.Fatalf("stat screenshots dir: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("screenshots path is not a directory")
	}
}
