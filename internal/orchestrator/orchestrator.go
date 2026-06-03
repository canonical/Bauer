package orchestrator

import (
	"bauer/internal/config"
	"bauer/internal/copilotcli"
	"bauer/internal/gdocs"
	"bauer/internal/prompt"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

// OrchestrationResult contains all outputs from the orchestration flow.
type OrchestrationResult struct {
	// Extraction
	ExtractionResult   *gdocs.ProcessingResult
	ExtractionDuration time.Duration

	// Parse-only result (populated when ParseOnly is true)
	ParseResult *ParseResult

	// Prompt generation
	Chunks       []prompt.ChunkResult
	PlanDuration time.Duration

	// Only populated if not dry run
	CopilotOutputs  []copilotcli.ChunkOutput
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
type DefaultOrchestrator struct{}

// NewOrchestrator creates a new DefaultOrchestrator instance.
func NewOrchestrator() *DefaultOrchestrator {
	return &DefaultOrchestrator{}
}

// Execute runs the full pipeline: extraction, prompt generation, and optional Copilot execution.
// Accepts: Config and Context
// Returns: OrchestrationResult and error
func (o *DefaultOrchestrator) Execute(ctx context.Context, cfg *config.Config) (*OrchestrationResult, error) {
	startTime := time.Now()

	// 1. Initialize GDocs Client and extract from doc
	extractionStart := time.Now()
	gdocsClient, err := gdocs.NewClient(ctx, cfg.CredentialsPath)
	if err != nil {
		slog.Error("Failed to initialize Google Docs client",
			slog.String("error", err.Error()),
			slog.String("credentials_path", cfg.CredentialsPath),
		)
		return nil, fmt.Errorf("failed to initialize Google Docs client: %w", err)
	}

	// 2. Process Document
	result, err := gdocsClient.ProcessDocument(ctx, cfg.DocID)
	if err != nil {
		return nil, fmt.Errorf("failed to process document: %w", err)
	}
	extractionDuration := time.Since(extractionStart)

	// 3. Write extraction result to file
	outputJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal output", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to generate output JSON: %w", err)
	}
	outputFile := "bauer-doc-suggestions.json"
	err = os.WriteFile(outputFile, outputJSON, 0644)
	if err != nil {
		slog.Error("Failed to write output file", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to write output file: %w", err)
	}
	slog.Info("Extraction complete",
		slog.String("output_file", outputFile),
		slog.Duration("extraction_duration", extractionDuration),
	)

	// Parse-only mode: return immediately after extraction without generating chunks
	if cfg.ParseOnly {
		totalDuration := time.Since(startTime)
		fileMappings := buildFileMappings(result)
		simplifiedSuggestions := buildSimplifiedSuggestions(result.ActionableSuggestions, fileMappings)
		summary := buildParseResultSummary(simplifiedSuggestions, fileMappings)

		parseResult := &ParseResult{
			Metadata: ParseResultMetadata{
				DocumentTitle:      result.DocumentTitle,
				DocumentID:         result.DocumentID,
				TabID:              result.TabID,
				ExtractionTime:     time.Now(),
				ExtractionDuration: extractionDuration,
				ProcessingDuration: totalDuration,
				TotalDuration:      totalDuration,
			},
			DocumentMetadata:      result.Metadata,
			Summary:               summary,
			FileMappings:          fileMappings,
			ActionableSuggestions: simplifiedSuggestions,
			GroupedSuggestions:    result.GroupedSuggestions,
			Comments:              result.Comments,
		}

		slog.Info("Parse-only mode: returning early after extraction",
			slog.Int("suggestion_count", len(simplifiedSuggestions)),
			slog.Int("file_count", len(fileMappings)),
			slog.Duration("total_duration", totalDuration),
		)

		return &OrchestrationResult{
			ExtractionResult:   result,
			ExtractionDuration: extractionDuration,
			ParseResult:        parseResult,
			Chunks:             []prompt.ChunkResult{},
			PlanDuration:       0,
			CopilotOutputs:     []copilotcli.ChunkOutput{},
			CopilotDuration:    0,
			SummaryDuration:    0,
			TotalDuration:      totalDuration,
			DryRun:             false,
		}, nil
	}

	// 4. Initialize Prompt Engine
	planStart := time.Now()
	engine, err := prompt.NewEngine(cfg.PageRefresh)
	if err != nil {
		slog.Error("Failed to initialize prompt engine", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to initialize prompt engine: %w", err)
	}

	// 5. Generate Prompts from Chunks
	totalLocations := len(result.GroupedSuggestions)
	slog.Info("Generating prompts",
		slog.Int("total_locations", totalLocations),
		slog.Int("chunk_size", cfg.ChunkSize),
	)
	chunks, err := engine.GenerateAllChunks(
		result,
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
	if cfg.DryRun {
		totalDuration := time.Since(startTime)

		return &OrchestrationResult{
			ExtractionResult:   result,
			ExtractionDuration: extractionDuration,
			Chunks:             chunks,
			PlanDuration:       planDuration,
			CopilotOutputs:     []copilotcli.ChunkOutput{},
			CopilotDuration:    0,
			SummaryDuration:    0,
			TotalDuration:      totalDuration,
			DryRun:             true,
		}, nil
	}

	// 6. Execute via Copilot SDK
	cwd, err := os.Getwd()
	if err != nil {
		slog.Error("Failed to get working directory", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	slog.Info("Initializing Copilot client", slog.String("cwd", cwd))
	copilotClient, err := copilotcli.NewClient(cwd)
	if err != nil {
		slog.Error("Failed to create Copilot client", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to create Copilot client: %w", err)
	}

	// Start the Copilot CLI server once
	if err := copilotClient.Start(); err != nil {
		// Attempt to stop the client if Start failed
		if stopErr := copilotClient.Stop(); stopErr != nil {
			slog.Error("Failed to stop Copilot client after start failure", slog.String("error", stopErr.Error()))
		}
		slog.Error("Failed to start Copilot", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to start Copilot: %w", err)
	}
	defer func() {
		if err := copilotClient.Stop(); err != nil {
			slog.Error("Failed to stop Copilot client", slog.String("error", err.Error()))
		}
	}()

	// Execute chunks via Copilot SDK
	chunkOutputs, copilotDuration, err := executeCopilotChunks(ctx, chunks, cfg, copilotClient)
	if err != nil {
		slog.Error("Copilot execution failed", slog.String("error", err.Error()))
		return nil, fmt.Errorf("copilot execution failed: %w", err)
	}

	slog.Info("Copilot chunks executed",
		slog.Int("chunk_count", len(chunks)),
		slog.Duration("total_duration", copilotDuration),
	)

	// 7. Generate summary if multiple chunks
	summaryDuration := time.Duration(0)
	if len(chunks) > 1 {
		summaryStart := time.Now()

		if err := copilotClient.GenerateSummary(ctx, chunkOutputs, cfg.SummaryModel); err != nil {
			slog.Error("Summary generation failed", slog.String("error", err.Error()))
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
		ExtractionResult:   result,
		ExtractionDuration: extractionDuration,
		Chunks:             chunks,
		PlanDuration:       planDuration,
		CopilotOutputs:     chunkOutputs,
		CopilotDuration:    copilotDuration,
		SummaryDuration:    summaryDuration,
		TotalDuration:      totalDuration,
		DryRun:             false,
	}, nil
}

// executeCopilotChunks executes each chunk via the Copilot SDK and returns outputs
func executeCopilotChunks(
	ctx context.Context,
	chunks []prompt.ChunkResult,
	cfg *config.Config,
	client *copilotcli.Client,
) ([]copilotcli.ChunkOutput, time.Duration, error) {
	executionStart := time.Now()

	var outputs []copilotcli.ChunkOutput
	totalChunks := len(chunks)

	for i, chunk := range chunks {
		chunkStart := time.Now()

		slog.Info("Executing chunk",
			slog.Int("chunk_number", chunk.ChunkNumber),
			slog.Int("chunk_count", totalChunks),
		)

		// Execute the chunk
		output, err := client.ExecuteChunk(ctx, chunk.Filename, chunk.ChunkNumber, cfg.Model)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to execute chunk %d: %w", chunk.ChunkNumber, err)
		}

		chunkDuration := time.Since(chunkStart)

		// Collect output
		outputs = append(outputs, copilotcli.ChunkOutput{
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
