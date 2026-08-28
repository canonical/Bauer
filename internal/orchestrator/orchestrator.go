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

	"github.com/google/uuid"
)

// OrchestrationResult contains all outputs from the orchestration flow.
type OrchestrationResult struct {
	// Extraction
	ExtractionResult   *gdocs.ProcessingResult
	ExtractionDuration time.Duration

	// Parse-only result (populated when ParseOnly is true)
	ParseResult *ParseResult

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
	requestID, _ := ctx.Value("requestID").(string)
	if requestID == "" {
		requestID = uuid.NewString()
	}

	// 1. Initialize GDocs Client and extract from doc
	extractionStart := time.Now()
	credentials, err := cfg.ResolveCredentials()
	if err != nil {
		slog.Error("Failed to resolve Google credentials", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to resolve Google credentials: %w", err)
	}
	gdocsClient, err := gdocs.NewClient(ctx, credentials)
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
	outputFile := fmt.Sprintf("bauer-doc-suggestions-%s.json", requestID)
	err = os.WriteFile(outputFile, outputJSON, 0644)
	if err != nil {
		slog.Error("Failed to write output file", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to write output file: %w", err)
	}
	slog.Info("Extraction complete",
		slog.String("output_file", outputFile),
		slog.Duration("extraction_duration", extractionDuration),
	)

	// Parse-only mode: return immediately after extraction without generating a prompt
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

	// 5. Generate Prompt
	totalLocations := len(result.GroupedSuggestions)
	slog.Info("Generating prompt",
		slog.Int("total_locations", totalLocations),
	)
	promptResult, err := engine.GeneratePrompt(
		result,
		cfg.OutputDir,
	)
	if err != nil {
		slog.Error("Failed to generate prompt", slog.String("error", err.Error()))
		return nil, fmt.Errorf("failed to generate prompt: %w", err)
	}

	planDuration := time.Since(planStart)

	slog.Info("Generated prompt",
		slog.String("filename", promptResult.Filename),
		slog.Int("location_count", promptResult.LocationCount),
	)

	totalDuration := time.Since(startTime)

	return &OrchestrationResult{
		ExtractionResult:   result,
		ExtractionDuration: extractionDuration,
		PlanDuration:       planDuration,
		TotalDuration:      totalDuration,
		ParseOnly:          false,
	}, nil
}
