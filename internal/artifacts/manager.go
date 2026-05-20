package artifacts

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"bauer/internal/figma"
	"bauer/internal/source/mapping"
)

// RunMetadata is written to metadata.json inside each run directory.
type RunMetadata struct {
	RunID        string `json:"run_id"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at,omitempty"`
	Status       string `json:"status"` // "in_progress", "success", "failed"
	DocID        string `json:"doc_id"`
	FigmaURL     string `json:"figma_url,omitempty"`
	FigmaVersion string `json:"figma_version,omitempty"`
	Mode         string `json:"mode"` // "execute", "dry-run", "issue"
	ChunkCount   int    `json:"chunk_count"`
	ArtifactDir  string `json:"artifact_dir"`
}

// RunIndexEntry is one line in runs.jsonl.
type RunIndexEntry struct {
	RunID        string `json:"run_id"`
	StartedAt    string `json:"started_at"`
	CompletedAt  string `json:"completed_at,omitempty"`
	Status       string `json:"status"`
	DocID        string `json:"doc_id"`
	FigmaURL     string `json:"figma_url,omitempty"`
	FigmaVersion string `json:"figma_version,omitempty"`
	Mode         string `json:"mode"`
	ChunkCount   int    `json:"chunk_count"`
	ArtifactDir  string `json:"artifact_dir"`
}

// Manager handles append-only artifact storage for Bauer runs.
type Manager struct {
	base string // root artifacts directory, e.g. "./bauer-artifacts"
}

// NewManager creates a Manager for the given artifacts directory.
func NewManager(base string) *Manager {
	if base == "" {
		base = "./bauer-artifacts"
	}
	return &Manager{base: base}
}

// NewRunID generates a unique run ID in the format YYYY-MM-DDTHH-MM-SSZ-{8hex}.
func NewRunID() string {
	now := time.Now().UTC()
	ts := now.Format("2006-01-02T15-04-05Z")
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s-%s", ts, hex.EncodeToString(b))
}

// StartRun creates the run directory structure and writes metadata.json with status "in_progress".
// Returns the runID for use in all subsequent calls.
func (m *Manager) StartRun(meta RunMetadata) (string, error) {
	if meta.RunID == "" {
		meta.RunID = NewRunID()
	}
	meta.Status = "in_progress"
	if meta.StartedAt == "" {
		meta.StartedAt = time.Now().UTC().Format(time.RFC3339)
	}
	if meta.ArtifactDir == "" {
		meta.ArtifactDir = filepath.Join(m.base, meta.RunID)
	}

	dirs := []string{
		filepath.Join(m.base, meta.RunID, "extraction"),
		filepath.Join(m.base, meta.RunID, "prompts"),
		filepath.Join(m.base, meta.RunID, "outputs"),
		filepath.Join(m.base, meta.RunID, "logs"),
		filepath.Join(m.base, meta.RunID, "screenshots"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return "", fmt.Errorf("create run dir %s: %w", d, err)
		}
	}

	if err := m.writeJSON(meta.RunID, "metadata.json", meta); err != nil {
		return "", err
	}

	return meta.RunID, nil
}

// CompleteRun updates metadata.json with the final status and appends to runs.jsonl.
func (m *Manager) CompleteRun(runID string, status string, chunkCount int) error {
	// Read existing metadata
	metaPath := filepath.Join(m.base, runID, "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("read metadata: %w", err)
	}
	var meta RunMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("parse metadata: %w", err)
	}
	meta.Status = status
	meta.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	meta.ChunkCount = chunkCount

	if err := m.writeJSON(runID, "metadata.json", meta); err != nil {
		return err
	}

	// Append to runs.jsonl
	entry := RunIndexEntry{
		RunID:        meta.RunID,
		StartedAt:    meta.StartedAt,
		CompletedAt:  meta.CompletedAt,
		Status:       status,
		DocID:        meta.DocID,
		FigmaURL:     meta.FigmaURL,
		FigmaVersion: meta.FigmaVersion,
		Mode:         meta.Mode,
		ChunkCount:   chunkCount,
		ArtifactDir:  meta.ArtifactDir,
	}
	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal run index entry: %w", err)
	}

	indexPath := filepath.Join(m.base, "runs.jsonl")
	f, err := os.OpenFile(indexPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open runs.jsonl: %w", err)
	}
	defer f.Close()
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

// WriteGDocsExtraction writes the gdocs extraction result to extraction/gdocs.json.
func (m *Manager) WriteGDocsExtraction(runID string, data any) error {
	return m.writeJSON(runID, "extraction/gdocs.json", data)
}

// WritePrompt writes a chunk prompt to prompts/chunk-N-of-M.md.
func (m *Manager) WritePrompt(runID string, chunkNum, totalChunks int, content string) error {
	filename := fmt.Sprintf("prompts/chunk-%d-of-%d.md", chunkNum, totalChunks)
	return m.writeFile(runID, filename, []byte(content))
}

// WriteOutput writes a chunk output to outputs/chunk-N-output.md.
func (m *Manager) WriteOutput(runID string, chunkNum int, content string) error {
	filename := fmt.Sprintf("outputs/chunk-%d-output.md", chunkNum)
	return m.writeFile(runID, filename, []byte(content))
}

// WriteSummary writes the summary output to outputs/summary.md.
func (m *Manager) WriteSummary(runID string, content string) error {
	return m.writeFile(runID, "outputs/summary.md", []byte(content))
}

// WriteIssueBody writes the formatted issue body to outputs/issue-body.md.
func (m *Manager) WriteIssueBody(runID string, content string) error {
	return m.writeFile(runID, "outputs/issue-body.md", []byte(content))
}

// EnsureScreenshotsDir ensures the screenshots directory exists and returns its path.
func (m *Manager) EnsureScreenshotsDir(runID string) (string, error) {
	dir := filepath.Join(m.base, runID, "screenshots")
	return dir, os.MkdirAll(dir, 0o755)
}

// WriteFigmaExtraction persists the normalized design to extraction/figma.json.
func (m *Manager) WriteFigmaExtraction(runID string, design *figma.NormalizedDesign) error {
	return m.writeJSON(runID, filepath.Join("extraction", "figma.json"), design)
}

// WriteMappings persists all resolved chunk mappings to extraction/mappings.json.
func (m *Manager) WriteMappings(runID string, chunks []mapping.ResolvedChunk) error {
	return m.writeJSON(runID, filepath.Join("extraction", "mappings.json"), chunks)
}

// WriteFigmaComments persists all extracted comments (including resolved) to extraction/comments.json.
func (m *Manager) WriteFigmaComments(runID string, comments []figma.DesignComment) error {
	return m.writeJSON(runID, filepath.Join("extraction", "comments.json"), comments)
}

// RunDir returns the path to the run's directory.
func (m *Manager) RunDir(runID string) string {
	return filepath.Join(m.base, runID)
}

// Base returns the root artifacts directory.
func (m *Manager) Base() string {
	return m.base
}

// LoadPreviousMeta returns the RunMetadata from the most recent completed run
// that matches the given (docID, figmaFileKey) pair, or nil if none exists.
// It reads runs.jsonl in reverse order (last entry = most recent) to find the match.
func (m *Manager) LoadPreviousMeta(docID, figmaFileKey string) *RunMetadata {
	indexPath := filepath.Join(m.base, "runs.jsonl")
	f, err := os.Open(indexPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	// Collect matching entries (in file order; last = most recent).
	var matched []RunIndexEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		var entry RunIndexEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}
		if entry.Status != "success" {
			continue
		}
		if entry.DocID != docID {
			continue
		}
		// Match figmaFileKey against the stored FigmaURL.
		if entry.FigmaURL == "" {
			continue
		}
		ref, err := figma.ParseLink(entry.FigmaURL)
		if err != nil || ref.FileKey != figmaFileKey {
			continue
		}
		matched = append(matched, entry)
	}
	if len(matched) == 0 {
		return nil
	}

	// Last entry is the most recent run.
	best := matched[len(matched)-1]
	metaPath := filepath.Join(m.base, best.RunID, "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil
	}
	var meta RunMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil
	}
	return &meta
}

// LoadMappings returns the resolved chunks from a previous run's mappings.json.
// Returns nil if the artifact does not exist or cannot be parsed.
func (m *Manager) LoadMappings(runID string) []mapping.ResolvedChunk {
	path := filepath.Join(m.base, runID, "extraction", "mappings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var chunks []mapping.ResolvedChunk
	if err := json.Unmarshal(data, &chunks); err != nil {
		return nil
	}
	return chunks
}

// UpdateRunFigmaVersion patches the run's metadata.json with the given figma version string.
// This is called after a successful Figma fetch so future runs can use it for drift detection.
func (m *Manager) UpdateRunFigmaVersion(runID, figmaVersion string) error {
	metaPath := filepath.Join(m.base, runID, "metadata.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		return fmt.Errorf("read metadata for version update: %w", err)
	}
	var meta RunMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return fmt.Errorf("parse metadata for version update: %w", err)
	}
	meta.FigmaVersion = figmaVersion
	return m.writeJSON(runID, "metadata.json", meta)
}

func (m *Manager) writeJSON(runID, relPath string, data any) error {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal %s: %w", relPath, err)
	}
	return m.writeFile(runID, relPath, b)
}

func (m *Manager) writeFile(runID, relPath string, data []byte) error {
	fullPath := filepath.Join(m.base, runID, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return fmt.Errorf("mkdir for %s: %w", relPath, err)
	}
	return os.WriteFile(fullPath, data, 0o644)
}
