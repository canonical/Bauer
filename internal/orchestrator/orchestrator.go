package orchestrator

import (
	"bauer/internal/agent"
	"bauer/internal/artifacts"
	"bauer/internal/config"
	"bauer/internal/figma"
	"bauer/internal/prompt"
	"bauer/internal/source"
	"bauer/internal/source/mapping"
	"context"
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
	// RunID is the artifact run identifier, empty if artifact storage was unavailable.
	RunID string

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
	arts    *artifacts.Manager
}

// New creates a new DefaultOrchestrator with the given agent, source manager, and artifact manager.
func New(a agent.Agent, sources *source.Manager, arts *artifacts.Manager) *DefaultOrchestrator {
	return &DefaultOrchestrator{agent: a, sources: sources, arts: arts}
}

// Execute runs the full pipeline: extraction, prompt generation, and optional agent execution.
func (o *DefaultOrchestrator) Execute(ctx context.Context, cfg *config.Config) (*OrchestrationResult, error) {
	startTime := time.Now()

	// Determine run mode
	mode := "execute"
	if config.BoolVal(cfg.DryRun, false) {
		mode = "dry-run"
	}

	// Start artifact run
	runID, err := o.arts.StartRun(artifacts.RunMetadata{
		DocID:    cfg.DocID,
		FigmaURL: cfg.FigmaURL,
		Mode:     mode,
	})
	if err != nil {
		slog.Warn("Failed to start artifact run", slog.String("error", err.Error()))
		// Non-fatal: continue without artifacts
		runID = ""
	}

	completeRun := func(status string, chunkCount int) {
		if runID == "" {
			return
		}
		if err := o.arts.CompleteRun(runID, status, chunkCount); err != nil {
			slog.Warn("Failed to complete artifact run", slog.String("error", err.Error()))
		}
	}

	// 1. Fetch from source (Google Docs)
	extractionStart := time.Now()
	bundle, err := o.sources.Fetch(ctx, source.Request{DocID: cfg.DocID})
	if err != nil {
		slog.Error("Failed to fetch from source",
			slog.String("error", err.Error()),
			slog.String("doc_id", cfg.DocID),
		)
		completeRun("failed", 0)
		return nil, fmt.Errorf("failed to fetch from source: %w", err)
	}
	extractionDuration := time.Since(extractionStart)

	// 2. Write extraction result to artifact store
	if bundle.Document != nil {
		if runID != "" {
			if err := o.arts.WriteGDocsExtraction(runID, bundle.Document); err != nil {
				slog.Warn("Failed to write gdocs extraction artifact", slog.String("error", err.Error()))
			}
		}
		slog.Info("Extraction complete",
			slog.Duration("extraction_duration", extractionDuration),
		)
	}

	// 3. Initialize Prompt Engine
	planStart := time.Now()
	engine, err := prompt.NewEngine(config.BoolVal(cfg.PageRefresh, false))
	if err != nil {
		slog.Error("Failed to initialize prompt engine", slog.String("error", err.Error()))
		completeRun("failed", 0)
		return nil, fmt.Errorf("failed to initialize prompt engine: %w", err)
	}

	// 4. Generate Prompts from Chunks
	if bundle.Document == nil {
		completeRun("failed", 0)
		return nil, fmt.Errorf("no document available: DocID may be empty")
	}

	totalLocations := len(bundle.Document.GroupedSuggestions)
	slog.Info("Generating prompts",
		slog.Int("total_locations", totalLocations),
		slog.Int("chunk_size", cfg.ChunkSize),
	)

	// When a Figma URL is configured, fetch design data and use figma-aware prompt generation.
	var chunks []prompt.ChunkResult
	if cfg.FigmaURL != "" {
		chunks, err = o.generateChunksWithFigma(ctx, cfg, bundle, engine, runID)
	} else {
		chunks, err = engine.GenerateAllChunks(
			bundle.Document,
			cfg.ChunkSize,
			cfg.OutputDir,
		)
	}
	if err != nil {
		slog.Error("Failed to generate prompts", slog.String("error", err.Error()))
		completeRun("failed", 0)
		return nil, fmt.Errorf("failed to generate prompts: %w", err)
	}

	planDuration := time.Since(planStart)

	// Write prompts to artifact store
	if runID != "" {
		for _, chunk := range chunks {
			// Read the generated prompt file and archive it
			if data, readErr := os.ReadFile(chunk.Filename); readErr == nil {
				if writeErr := o.arts.WritePrompt(runID, chunk.ChunkNumber, len(chunks), string(data)); writeErr != nil {
					slog.Warn("Failed to write prompt artifact", slog.String("error", writeErr.Error()))
				}
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
		totalDuration := time.Since(startTime)
		completeRun("success", len(chunks))
		return &OrchestrationResult{
			RunID:              runID,
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
		completeRun("failed", len(chunks))
		return nil, fmt.Errorf("agent execution failed: %w", err)
	}

	// Write chunk outputs to artifact store
	if runID != "" {
		for _, co := range chunkOutputs {
			if writeErr := o.arts.WriteOutput(runID, co.ChunkNumber, co.Output); writeErr != nil {
				slog.Warn("Failed to write chunk output artifact", slog.String("error", writeErr.Error()))
			}
		}
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
			if runID != "" {
				if writeErr := o.arts.WriteSummary(runID, summary); writeErr != nil {
					slog.Warn("Failed to write summary artifact", slog.String("error", writeErr.Error()))
				}
			}
		}
	}

	totalDuration := time.Since(startTime)
	completeRun("success", len(chunks))

	return &OrchestrationResult{
		RunID:              runID,
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

// generateChunksWithFigma fetches Figma design data and produces figma-aware prompt files.
// It is called by Execute when cfg.FigmaURL is non-empty.
func (o *DefaultOrchestrator) generateChunksWithFigma(
	ctx context.Context,
	cfg *config.Config,
	bundle *source.SourceBundle,
	engine *prompt.Engine,
	runID string,
) ([]prompt.ChunkResult, error) {
	figmaRef, err := figma.ParseLink(cfg.FigmaURL)
	if err != nil {
		return nil, fmt.Errorf("invalid figma URL %q: %w", cfg.FigmaURL, err)
	}

	figmaClient := figma.NewClient(cfg.FigmaToken)

	// Determine screenshot directory (inside the artifact run when available).
	screenshotDir := ""
	if runID != "" {
		screenshotDir, err = o.arts.EnsureScreenshotsDir(runID)
		if err != nil {
			slog.Warn("Failed to create screenshots dir, proceeding without screenshots",
				slog.String("error", err.Error()))
			screenshotDir = ""
		}
	}

	if screenshotDir == "" {
		slog.Warn("Screenshot directory unavailable, skipping screenshot download")
	}

	slog.Info("Fetching Figma design data", slog.String("figma_url", cfg.FigmaURL))
	design, err := o.sources.FetchFigma(ctx, figmaClient, figmaRef, screenshotDir)
	if err != nil {
		return nil, fmt.Errorf("fetching figma design: %w", err)
	}
	slog.Info("Figma design fetched",
		slog.Int("anchors", len(design.Anchors)),
		slog.Int("comments", len(design.Comments)),
	)

	// Persist figma artifacts.
	if runID != "" {
		if werr := o.arts.WriteFigmaExtraction(runID, design); werr != nil {
			slog.Warn("Failed to write figma extraction artifact", slog.String("error", werr.Error()))
		}
		if werr := o.arts.WriteFigmaComments(runID, design.Comments); werr != nil {
			slog.Warn("Failed to write figma comments artifact", slog.String("error", werr.Error()))
		}
	}

	// Build resolved chunks (design-aware mapping).
	resolver := &mapping.Resolver{}
	resolvedChunks := resolver.Build(bundle.Document.GroupedSuggestions, design)
	slog.Info("Mapping resolved", slog.Int("resolved_chunks", len(resolvedChunks)))

	if runID != "" {
		if werr := o.arts.WriteMappings(runID, resolvedChunks); werr != nil {
			slog.Warn("Failed to write mappings artifact", slog.String("error", werr.Error()))
		}
	}

	// Generate figma-aware prompt files.
	suggestedURL := ""
	if bundle.Document.Metadata != nil {
		suggestedURL = bundle.Document.Metadata.SuggestedUrl
	}

	return engine.RenderChunksFromResolved(
		bundle.Document.DocumentTitle,
		suggestedURL,
		cfg.FigmaURL,
		resolvedChunks,
		cfg.ChunkSize,
		cfg.OutputDir,
	)
}

// executeAgentChunks executes each chunk via the agent and returns outputs.
func executeAgentChunks(ctx context.Context,
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
