// Package artifacts manages append-only run artifact history. Each run gets a
// timestamped directory under the configured artifacts dir. A runs.jsonl index
// file receives one line per completed run and is never rewritten in full.
package artifacts

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// RunMeta is written as one line in runs.jsonl after each completed run.
type RunMeta struct {
	RunID       string `json:"run_id"`
	StartedAt   string `json:"started_at"`
	CompletedAt string `json:"completed_at"`
	Status      string `json:"status"` // "success" | "failed" | "in_progress"
	DocID       string `json:"doc_id"`
	FigmaURL    string `json:"figma_url"`
	Mode        string `json:"mode"` // "execute" | "dry-run" | "issue"
	ChunkCount  int    `json:"chunk_count"`
	ArtifactDir string `json:"artifact_dir"`
}

// Manager handles writing run artifacts to the filesystem.
type Manager struct {
	base string
	mu   sync.Mutex
}

// NewManager creates a Manager rooted at baseDir.
func NewManager(baseDir string) *Manager {
	return &Manager{base: baseDir}
}

// BaseDir returns the configured artifacts base directory.
func (m *Manager) BaseDir() string {
	return m.base
}

// NewRun creates a new timestamped run directory and returns the run ID.
// It also creates the standard subdirectories (extraction, prompts, outputs, logs).
func (m *Manager) NewRun(ctx context.Context) (string, error) {
	_ = ctx

	if err := os.MkdirAll(m.base, 0o755); err != nil {
		return "", fmt.Errorf("create artifacts dir: %w", err)
	}

	now := time.Now().UTC()
	runID := fmt.Sprintf("%s-%04x", now.Format("2006-01-02T15-04-05Z"), now.Nanosecond()/65536)
	runDir := filepath.Join(m.base, runID)

	dirs := []string{
		filepath.Join(runDir, "extraction"),
		filepath.Join(runDir, "prompts"),
		filepath.Join(runDir, "outputs"),
		filepath.Join(runDir, "logs"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return "", fmt.Errorf("create %s: %w", d, err)
		}
	}

	return runID, nil
}

// RunDir returns the absolute path for a given run ID.
func (m *Manager) RunDir(runID string) string {
	return filepath.Join(m.base, runID)
}

// AppendRun appends a completed run entry to runs.jsonl. The file is created
// on first write and only ever appended to.
func (m *Manager) AppendRun(meta RunMeta) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := os.MkdirAll(m.base, 0o755); err != nil {
		return fmt.Errorf("create artifacts dir: %w", err)
	}

	path := filepath.Join(m.base, "runs.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open runs.jsonl: %w", err)
	}
	defer f.Close()

	line, err := json.Marshal(meta)
	if err != nil {
		return fmt.Errorf("marshal run meta: %w", err)
	}

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write runs.jsonl: %w", err)
	}

	return nil
}

// WriteMetadata writes a metadata.json file inside the run directory.
func (m *Manager) WriteMetadata(runID string, meta RunMeta) error {
	return m.writeJSON(runID, "metadata.json", meta)
}

// WriteExtraction writes an extraction file to the run's extraction directory.
func (m *Manager) WriteExtraction(runID string, name string, data any) error {
	return m.writeJSON(runID, filepath.Join("extraction", name), data)
}

// WritePrompt writes a prompt file to the run's prompts directory.
func (m *Manager) WritePrompt(runID string, name string, content string) error {
	runDir := filepath.Join(m.base, runID, "prompts")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("create prompts dir: %w", err)
	}
	path := filepath.Join(runDir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write prompt %s: %w", name, err)
	}
	return nil
}

// WriteOutput writes an output file to the run's outputs directory.
func (m *Manager) WriteOutput(runID string, name string, content string) error {
	runDir := filepath.Join(m.base, runID, "outputs")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("create outputs dir: %w", err)
	}
	path := filepath.Join(runDir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write output %s: %w", name, err)
	}
	return nil
}

// LogEntry is a single structured log line written to execution.jsonl.
type LogEntry struct {
	Timestamp string `json:"timestamp"`
	Step      string `json:"step"`
	Message   string `json:"message"`
}

// AppendLog appends a structured log entry to the run's logs/execution.jsonl.
func (m *Manager) AppendLog(runID string, entry LogEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	runDir := filepath.Join(m.base, runID, "logs")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return fmt.Errorf("create logs dir: %w", err)
	}

	path := filepath.Join(runDir, "execution.jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open execution.jsonl: %w", err)
	}
	defer f.Close()

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal log entry: %w", err)
	}

	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("write execution.jsonl: %w", err)
	}

	return nil
}

// EnsureScreenshotsDir creates and returns the screenshots directory for a run.
func (m *Manager) EnsureScreenshotsDir(runID string) (string, error) {
	dir := filepath.Join(m.base, runID, "screenshots")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create screenshots dir: %w", err)
	}
	return dir, nil
}

// writeJSON is a helper to write any value as pretty-printed JSON to a run file.
func (m *Manager) writeJSON(runID, relPath string, data any) error {
	fullPath := filepath.Join(m.base, runID, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("create dir for %s: %w", relPath, err)
	}

	raw, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", relPath, err)
	}

	if err := os.WriteFile(fullPath, raw, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", relPath, err)
	}

	slog.Debug("Artifact written", "path", fullPath)
	return nil
}
