package orchestrator

import (
	"bauer/internal/agent"
	"bauer/internal/config"
	"bauer/internal/prompt"
	"bauer/internal/source"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// ChunkOutput represents the output from a single agent chunk execution.
type ChunkOutput struct {
	ChunkNumber int
	Output      string
	Duration    time.Duration
}

// OrchestrationResult contains all outputs from the orchestration flow.
type OrchestrationResult struct {
	// Extraction
	ExtractionBundle   *source.SourceBundle
	ExtractionDuration time.Duration

	// Prompt generation
	Chunks       []prompt.ChunkResult
	PlanDuration time.Duration

	// Only populated if not dry run
	CopilotOutputs  []ChunkOutput
	Summary         string
	CopilotDuration time.Duration
	SummaryDuration time.Duration

	// Metadata
	TotalDuration time.Duration
	DryRun        bool
}

// Orchestrator defines the interface for executing the BAU orchestration flow.
type Orchestrator interface {
	Execute(ctx context.Context, cfg *config.Config) (*OrchestrationResult, error)
}

// DefaultOrchestrator is the standard implementation of the Orchestrator interface.
type DefaultOrchestrator struct {
	agent   agent.Agent
	sources *source.Manager
}

// New creates a new DefaultOrchestrator with the given agent and source manager.
func New(a agent.Agent, sources *source.Manager) *DefaultOrchestrator {
	return &DefaultOrchestrator{agent: a, sources: sources}
}

// Execute runs the full pipeline: extraction, prompt generation, and optional agent execution.
func (o *DefaultOrchestrator) Execute(ctx context.Context, cfg *config.Config) (*OrchestrationResult, error) {
	startTime := time.Now()

	// 1. Fetch from source (Google Docs)
	extractionStart := time.Now()
	bundle, err := o.sources.Fetch(ctx, source.Request{DocID: cfg.DocID})
	if err != nil {
		slog.Error("Failed to fetch from source",
			slog.String("error", err.Error()),
			slog.String("doc_id", cfg.DocID),
		)
		return nil, fmt.Errorf("failed to fetch from source: %w", err)
	}
	extractionDuration := time.Since(extractionStart)

	// 2. Write extraction result to file
	if bundle.Document != nil {
		outputJSON, err := json.MarshalIndent(bundle.Document, "", "  ")
		if err != nil {
			slog.Error("Failed to marshal output", slog.String("error", err.Error()))
			return nil, fmt.Errorf("failed to generate output JSON: %w", err)
		}
		outputFile := "bauer-doc-suggestions.json"
		if err = os.WriteFile(outputFile, outputJSON, 0644); err != nil {
			slog.Error("Failed to write output file", slog.String("error", err.Error()))
			return nil, fmt.Errorf("failed to write output file: %w", err)
		}
		slog.Info("Extraction complete",
			slog.String("output_file", outputFile),
			slog.Duration("extraction_duration", extractionDuration),
		)
	}

	// 3. Initialize Prompt Engine
	planStart := time.Now()
	engine, err := prompt.NewEngine(cfg.PageRefresh)
	if err != nil {
		slog.Error("Failed to initialize prompt engine", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to initialize prompt engine: %w", err)
	}

	// 4. Generate Prompts from Chunks
	if bundle.Document == nil {
		return nil, fmt.Errorf("no document available: DocID may be empty")
	}

	totalLocations := len(bundle.Document.GroupedSuggestions)
	slog.Info("Generating prompts",
		slog.Int("total_locations", totalLocations),
		slog.Int("chunk_size", cfg.ChunkSize),
	)
	chunks, err := engine.GenerateAllChunks(
		bundle.Document,
		cfg.ChunkSize,
		cfg.OutputDir,
	)
	if err != nil {
		slog.Error("Failed to generate prompts", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to generate prompts: %w", err)
	}

	planDuration := time.Since(planStart)

	for _, chunk := range chunks {
		slog.Info("Generated chunk",
			slog.Int("chunk_number", chunk.ChunkNumber),
			slog.String("filename", chunk.Filename),
			slog.Int("location_count", chunk.LocationCount),
		)
	}

	// If dry run, return early
	if config.BoolVal(cfg.DryRun, false) {
		totalDuration := time.Since(startTime)
		return &OrchestrationResult{
			ExtractionBundle:   bundle,
			ExtractionDuration: extractionDuration,
			Chunks:             chunks,
			PlanDuration:       planDuration,
			CopilotOutputs:     []ChunkOutput{},
			CopilotDuration:    0,
			SummaryDuration:    0,
			TotalDuration:      totalDuration,
			DryRun:             true,
		}, nil
	}

	// 5. Start agent
	slog.Info("Starting agent")
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

	// 6. Execute chunks via agent
	chunkOutputs, copilotDuration, err := executeAgentChunks(ctx, chunks, cfg, o.agent)
	if err != nil {
		slog.Error("Agent execution failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	slog.Info("Agent chunks executed",
		slog.Int("chunk_count", len(chunks)),
		slog.Duration("total_duration", copilotDuration),
	)

	// 7. Generate summary if multiple chunks
	summaryDuration := time.Duration(0)
	var summary string
	if len(chunks) > 1 {
		summaryStart := time.Now()

		outputs := make([]string, len(chunkOutputs))
		for i, co := range chunkOutputs {
			outputs[i] = co.Output
		}

		var summaryErr error
		summary, summaryErr = o.agent.GenerateSummary(ctx, outputs, cfg.SummaryModel)
		if summaryErr != nil {
			slog.Error("Summary generation failed", slog.String("error", summaryErr.Error()))
			// Summary failure is not fatal; continue with results
		} else {
			summaryDuration = time.Since(summaryStart)
			slog.Info("Summary generated successfully",
				slog.Duration("duration", summaryDuration),
			)
		}
	}

	totalDuration := time.Since(startTime)

	return &OrchestrationResult{
		ExtractionBundle:   bundle,
		ExtractionDuration: extractionDuration,
		Chunks:             chunks,
		PlanDuration:       planDuration,
		CopilotOutputs:     chunkOutputs,
		Summary:            summary,
		CopilotDuration:    copilotDuration,
		SummaryDuration:    summaryDuration,
		TotalDuration:      totalDuration,
		DryRun:             false,
	}, nil
}

// executeAgentChunks executes each chunk via the agent and returns outputs.
func executeAgentChunks(
	ctx context.Context,
	chunks []prompt.ChunkResult,
	cfg *config.Config,
	a agent.Agent,
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

		output, err := a.ExecuteChunk(ctx, chunk.Filename, chunk.ChunkNumber, cfg.Model)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to execute chunk %d: %w", chunk.ChunkNumber, err)
		}

		chunkDuration := time.Since(chunkStart)
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
