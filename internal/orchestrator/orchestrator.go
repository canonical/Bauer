package orchestrator

import (
	"bauer/internal/artifacts"
	"bauer/internal/config"
	"bauer/internal/prompt"
	"bauer/internal/source"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// ChunkOutput holds the result of a single chunk execution.
type ChunkOutput struct {
	ChunkNumber int
	Output      string
	Duration    time.Duration
}

// OrchestrationResult contains all outputs from the orchestration flow.
type OrchestrationResult struct {
	// Source bundle from the source layer
	Bundle *source.SourceBundle

	// Run ID for the artifact directory
	RunID string

	// Extraction
	ExtractionDuration time.Duration

	// Prompt generation
	Chunks       []prompt.ChunkResult
	PlanDuration time.Duration

	// Only populated if not dry run
	CopilotOutputs  []ChunkOutput
	CopilotDuration time.Duration
	SummaryDuration time.Duration

	// Metadata
	TotalDuration time.Duration
	DryRun        bool
}

// Agent defines the execution contract for any AI backend used by the orchestrator.
// Defined here at the consumer following Go convention; implementations
// (copilotcli.Client, test mocks) satisfy it implicitly.
type Agent interface {
	// Start boots the agent (e.g. starts the Copilot SDK server process).
	// Must be called before any other method. Callers should defer Stop().
	Start(ctx context.Context) error

	// ExecuteChunk sends a single chunk prompt file to the agent and returns
	// the full text output. chunkNum is for logging/display only.
	ExecuteChunk(ctx context.Context, chunkPath string, chunkNum int, model string) (string, error)

	// GenerateSummary produces a summary of all chunk outputs.
	// Only called when there are multiple chunks.
	GenerateSummary(ctx context.Context, outputs []string, model string) (string, error)

	// Stop shuts the agent down cleanly. Safe to call after a failed Start.
	Stop() error
}

// Orchestrator defines the interface for executing the BAU orchestration flow.
type Orchestrator interface {
	Execute(ctx context.Context, cfg *config.Config) (*OrchestrationResult, error)
}

// DefaultOrchestrator is the standard implementation of the Orchestrator interface.
type DefaultOrchestrator struct {
	agent     Agent
	sources   *source.Manager
	artifacts *artifacts.Manager
}

// NewOrchestrator creates a new DefaultOrchestrator. Pass any Agent implementation,
// a source.Manager, and an artifacts.Manager.
func NewOrchestrator(a Agent, s *source.Manager, art *artifacts.Manager) *DefaultOrchestrator {
	return &DefaultOrchestrator{agent: a, sources: s, artifacts: art}
}

// SetAgent replaces the agent. Useful when the agent must be created after
// the orchestrator (e.g. after chdir to a cloned repo).
func (o *DefaultOrchestrator) SetAgent(a Agent) {
	o.agent = a
}

// Execute runs the full pipeline: source fetch, prompt generation, and optional agent execution.
func (o *DefaultOrchestrator) Execute(ctx context.Context, cfg *config.Config) (result *OrchestrationResult, retErr error) {
	startTime := time.Now()

	// Create artifact run directory
	var runID string
	if o.artifacts != nil {
		var err error
		runID, err = o.artifacts.NewRun(ctx)
		if err != nil {
			return nil, fmt.Errorf("create run directory: %w", err)
		}
		slog.Info("Created artifact run directory", "run_id", runID)
	}

	// Ensure finalizeRun is called even on error, so failed runs appear in history
	defer func() {
		if o.artifacts == nil || runID == "" {
			return
		}
		// Build result for finalize if we haven't set one yet
		if result == nil {
			result = &OrchestrationResult{RunID: runID}
		}
		status := "success"
		if retErr != nil {
			status = "failed"
		}
		mode := "execute"
		if config.BoolVal(cfg.DryRun, false) {
			mode = "dry-run"
		}
		o.finalizeRun(runID, result, cfg, mode, status)
	}()

	// 1. Fetch from all configured sources via the source layer
	extractionStart := time.Now()

	req := source.Request{
		DocID:           cfg.DocID,
		CredentialsPath: cfg.CredentialsPath,
		FigmaURL:        "", // Figma support added in spec 002
	}

	bundle, err := o.sources.Fetch(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("source fetch: %w", err)
	}

	if bundle.Document == nil {
		return nil, fmt.Errorf("source fetch returned no document data")
	}

	result = &OrchestrationResult{
		Bundle: bundle,
		RunID:  runID,
	}

	doc := bundle.Document
	extractionDuration := time.Since(extractionStart)
	result.ExtractionDuration = extractionDuration

	// 2. Write extraction result to artifact directory
	if o.artifacts != nil && runID != "" {
		if err := o.artifacts.WriteExtraction(runID, "gdocs.json", doc); err != nil {
			slog.Error("Failed to write extraction artifact", slog.String("error", err.Error()))
		}
	}

	// Also write to legacy output file for backward compatibility
	outputJSON, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal output", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to generate output JSON: %w", err)
	}
	outputFile := "bauer-doc-suggestions.json"
	if err := os.WriteFile(outputFile, outputJSON, 0644); err != nil {
		slog.Error("Failed to write output file", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to write output file: %w", err)
	}
	slog.Info("Extraction complete",
		slog.String("output_file", outputFile),
		slog.Duration("extraction_duration", extractionDuration),
	)

	// 3. Initialize Prompt Engine
	planStart := time.Now()
	engine, err := prompt.NewEngine(config.BoolVal(cfg.PageRefresh, false))
	if err != nil {
		slog.Error("Failed to initialize prompt engine", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to initialize prompt engine: %w", err)
	}

	// 4. Generate Prompts from Chunks
	totalLocations := len(doc.GroupedSuggestions)
	slog.Info("Generating prompts",
		slog.Int("total_locations", totalLocations),
		slog.Int("chunk_size", cfg.ChunkSize),
	)
	chunks, err := engine.GenerateAllChunks(
		doc,
		cfg.ChunkSize,
		cfg.OutputDir,
	)
	if err != nil {
		slog.Error("Failed to generate prompts", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to generate prompts: %w", err)
	}

	planDuration := time.Since(planStart)
	result.PlanDuration = planDuration
	result.Chunks = chunks

	// Write prompt artifacts (use base filename only)
	if o.artifacts != nil && runID != "" {
		for _, chunk := range chunks {
			artifactName := filepath.Base(chunk.Filename)
			if err := o.artifacts.WritePrompt(runID, artifactName, chunk.Content); err != nil {
				slog.Error("Failed to write prompt artifact", slog.String("error", err.Error()))
			}
		}
	}

	for _, chunk := range chunks {
		slog.Info("Generated chunk",
			slog.Int("chunk_number", chunk.ChunkNumber),
			slog.String("filename", chunk.Filename),
			slog.Int("location_count", chunk.LocationCount),
		)
	}

	// If dry run, return early
	if config.BoolVal(cfg.DryRun, false) {
		result.CopilotOutputs = []ChunkOutput{}
		result.DryRun = true
		result.TotalDuration = time.Since(startTime)
		return result, nil
	}

	// 5. Execute via configured agent
	if o.agent == nil {
		return nil, fmt.Errorf("agent is required")
	}

	if err := o.agent.Start(ctx); err != nil {
		if stopErr := o.agent.Stop(); stopErr != nil {
			slog.Error("Failed to stop agent after start failure", slog.String("error", stopErr.Error()))
		}
		slog.Error("Failed to start agent", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to start agent: %w", err)
	}
	defer func() {
		if err := o.agent.Stop(); err != nil {
			slog.Error("Failed to stop agent", slog.String("error", err.Error()))
		}
	}()

	// Execute chunks via agent
	chunkOutputs, copilotDuration, err := executeCopilotChunks(ctx, chunks, cfg, o.agent)
	if err != nil {
		slog.Error("Copilot execution failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("copilot execution failed: %w", err)
	}

	// Write output artifacts
	if o.artifacts != nil && runID != "" {
		for _, output := range chunkOutputs {
			name := fmt.Sprintf("chunk-%d-output.md", output.ChunkNumber)
			if err := o.artifacts.WriteOutput(runID, name, output.Output); err != nil {
				slog.Error("Failed to write output artifact", slog.String("error", err.Error()))
			}
		}
	}

	result.CopilotOutputs = chunkOutputs
	result.CopilotDuration = copilotDuration

	slog.Info("Copilot chunks executed",
		slog.Int("chunk_count", len(chunks)),
		slog.Duration("total_duration", copilotDuration),
	)

	// 6. Generate summary if multiple chunks
	summaryDuration := time.Duration(0)
	if len(chunks) > 1 {
		summaryStart := time.Now()

		summaryInputs := make([]string, 0, len(chunkOutputs))
		for _, output := range chunkOutputs {
			summaryInputs = append(summaryInputs, output.Output)
		}

		summary, err := o.agent.GenerateSummary(ctx, summaryInputs, cfg.SummaryModel)
		if err != nil {
			slog.Error("Summary generation failed", slog.String("error", err.Error()))
		} else {
			summaryDuration = time.Since(summaryStart)
			slog.Info("Summary generated successfully",
				slog.Duration("duration", summaryDuration),
			)
			if o.artifacts != nil && runID != "" && summary != "" {
				if err := o.artifacts.WriteOutput(runID, "summary.md", summary); err != nil {
					slog.Error("Failed to write summary artifact", slog.String("error", err.Error()))
				}
			}
		}
	}
	result.SummaryDuration = summaryDuration
	result.TotalDuration = time.Since(startTime)

	return result, nil
}

// finalizeRun writes metadata and appends to runs.jsonl.
func (o *DefaultOrchestrator) finalizeRun(runID string, res *OrchestrationResult, cfg *config.Config, mode, status string) {
	if o.artifacts == nil || runID == "" {
		return
	}

	artifactDir := filepath.Join(o.artifacts.BaseDir(), runID)

	meta := artifacts.RunMeta{
		RunID:       runID,
		StartedAt:   time.Now().Add(-res.TotalDuration).UTC().Format(time.RFC3339),
		CompletedAt: time.Now().UTC().Format(time.RFC3339),
		Status:      status,
		DocID:       cfg.DocID,
		Mode:        mode,
		ChunkCount:  len(res.Chunks),
		ArtifactDir: artifactDir,
	}

	if err := o.artifacts.WriteMetadata(runID, meta); err != nil {
		slog.Error("Failed to write run metadata", slog.String("error", err.Error()))
	}

	if err := o.artifacts.AppendRun(meta); err != nil {
		slog.Error("Failed to append run to index", slog.String("error", err.Error()))
	}
}

// executeCopilotChunks executes each chunk via the configured agent and returns outputs
func executeCopilotChunks(
	ctx context.Context,
	chunks []prompt.ChunkResult,
	cfg *config.Config,
	executor Agent,
) ([]ChunkOutput, time.Duration, error) {
	executionStart := time.Now()

	var outputs []ChunkOutput
	totalChunks := len(chunks)

	for i, chunk := range chunks {
		chunkStart := time.Now()

		slog.Info("Executing chunk",
			slog.Int("chunk_number", chunk.ChunkNumber),
			slog.Int("chunk_count", totalChunks),
		)

		// Execute the chunk
		output, err := executor.ExecuteChunk(ctx, chunk.Filename, chunk.ChunkNumber, cfg.Model)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to execute chunk %d: %w", chunk.ChunkNumber, err)
		}

		chunkDuration := time.Since(chunkStart)

		// Collect output
		outputs = append(outputs, ChunkOutput{
			ChunkNumber: chunk.ChunkNumber,
			Output:      output,
			Duration:    chunkDuration,
		})

		slog.Info("Chunk executed successfully",
			slog.Int("chunk", chunk.ChunkNumber),
			slog.Int("completed", i+1),
			slog.Int("total", totalChunks),
			slog.Duration("duration", chunkDuration),
		)
	}

	totalDuration := time.Since(executionStart)
	return outputs, totalDuration, nil
}
