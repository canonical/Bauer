package orchestrator

import (
	"bauer/internal/config"
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

	// Metadata
	TotalDuration time.Duration
	ParseOnly     bool
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

// Execute runs the full pipeline: extraction, prompt generation, and optional GitHub integration.
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
			TotalDuration:      totalDuration,
			ParseOnly:          cfg.ParseOnly,
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

	totalDuration := time.Since(startTime)

	return &OrchestrationResult{
		ExtractionResult:   result,
		ExtractionDuration: extractionDuration,
		Chunks:             chunks,
		PlanDuration:       planDuration,
		TotalDuration:      totalDuration,
		ParseOnly:          false,
	}, nil
}
