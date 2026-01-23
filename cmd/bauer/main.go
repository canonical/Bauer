package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"bauer/internal/config"
	"bauer/internal/copilotcli"
	"bauer/internal/gdocs"
	"bauer/internal/prompt"
)

func main() {
	fmt.Println("BAU - Build Automation Utility")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// 2. Setup Logging
	// For now, mirroring POC behavior: logging to log.json in the current directory
	// TODO disable with a flag or env var
	logFile, err := os.OpenFile("bauer-log.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open log file: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close log file: %v\n", err)
		}
	}()

	logger := slog.New(slog.NewJSONHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	slog.Info("Starting BAU CLI",
		slog.String("doc_id", cfg.DocID),
		slog.Bool("dry_run", cfg.DryRun),
	)

	ctx := context.Background()

	// 3. Initialize GDocs Client
	fmt.Println("[1/4] Extracting from Google Doc...")
	client, err := gdocs.NewClient(ctx, cfg.CredentialsPath)
	if err != nil {
		slog.Error("Failed to initialize Google Docs client", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "Failed to initialize Google Docs client: %v\n", err)
		os.Exit(1)
	}

	// 4. Process Document
	result, err := client.ProcessDocument(ctx, cfg.DocID)
	if err != nil {
		// Error logging is handled in ProcessDocument
		os.Exit(1)
	}

	// 5. Generate Output JSON (for reference)
	outputJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		slog.Error("Failed to marshal output", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "Failed to generate output JSON: %v\n", err)
		os.Exit(1)
	}

	// Write JSON to file
	outputFile := "doc-suggestions.json"
	err = os.WriteFile(outputFile, outputJSON, 0644)
	if err != nil {
		slog.Error("Failed to write output file", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "Failed to write output file: %v\n", err)
		os.Exit(1)
	}

	slog.Info("Extraction complete", slog.String("output_file", outputFile))

	// 6. Initialize Prompt Engine
	fmt.Println("[2/4] Generating technical plan...")
	engine, err := prompt.NewEngine()
	if err != nil {
		slog.Error("Failed to initialize prompt engine", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "Failed to initialize prompt engine: %v\n", err)
		os.Exit(1)
	}

	// 7. Generate Prompts from Chunks
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
		fmt.Fprintf(os.Stderr, "Failed to generate prompts: %v\n", err)
		os.Exit(1)
	}

	// 8. Report Results
	fmt.Printf("  ✓ Saved: %s\n", outputFile)
	fmt.Printf("  ✓ Generated %d chunk file(s) in '%s/'\n", len(chunks), cfg.OutputDir)
	for _, chunk := range chunks {
		slog.Info("Generated chunk",
			slog.Int("chunk_number", chunk.ChunkNumber),
			slog.String("filename", chunk.Filename),
			slog.Int("location_count", chunk.LocationCount),
		)
	}
	fmt.Println()

	if cfg.DryRun {
		fmt.Println("[3/4] Copilot execution (skipped - dry run)")
		fmt.Println()
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println("DRY RUN COMPLETE")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Printf("  Summary:\n")
		fmt.Printf("    • Extracted: %d suggestions\n", len(result.ActionableSuggestions))
		fmt.Printf("    • Grouped into: %d location(s)\n", totalLocations)
		fmt.Printf("    • Generated: %d chunk file(s) in '%s/'\n", len(chunks), cfg.OutputDir)
		fmt.Printf("\n  Next steps:\n")
		fmt.Printf("    1. Review generated chunks in '%s/'\n", cfg.OutputDir)
		fmt.Printf("    2. Run without --dry-run to execute changes via Copilot\n")
		fmt.Printf("    3. Or manually pass chunk files to: gh copilot\n")
		fmt.Println(strings.Repeat("=", 80))
		return
	}

	// 9. Execute via Copilot SDK
	fmt.Println("[3/4] Executing changes via Copilot...")
	fmt.Println(strings.Repeat("=", 80))

	// Execute chunks via Copilot SDK
	if err := executeCopilotChunks(ctx, chunks, cfg); err != nil {
		slog.Error("Copilot execution failed", slog.String("error", err.Error()))
		fmt.Fprintf(os.Stderr, "\n❌ Copilot execution failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("  ✓ All %d chunk(s) processed successfully\n", len(chunks))
	fmt.Println()

	// 10. Next steps
	fmt.Println("[4/4] Next steps...")
	fmt.Println("  • Review the changes made by Copilot")
	fmt.Println("  • Create a PR with: gh pr create")
	fmt.Println()
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("SUCCESS: Feedback applied!")
	fmt.Println(strings.Repeat("=", 80))
}

// executeCopilotChunks executes each chunk via the Copilot SDK
func executeCopilotChunks(ctx context.Context, chunks []prompt.ChunkResult, cfg *config.Config) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Initialize Copilot client
	slog.Info("Initializing Copilot client", slog.String("cwd", cwd))
	client, err := copilotcli.NewClient(cwd)
	if err != nil {
		return fmt.Errorf("failed to create Copilot client: %w", err)
	}

	// Start the Copilot CLI server
	if err := client.Start(); err != nil {
		return fmt.Errorf("failed to start Copilot: %w", err)
	}
	defer func() {
		if err := client.Stop(); err != nil {
			slog.Error("Failed to stop Copilot client", slog.String("error", err.Error()))
		}
	}()

	// Execute each chunk sequentially
	totalChunks := len(chunks)
	for i, chunk := range chunks {
		fmt.Printf("  [Chunk %d/%d] Processing %s...\n", i+1, totalChunks, chunk.Filename)

		// Execute the chunk
		if err := client.ExecuteChunk(ctx, chunk.Filename, chunk.ChunkNumber, cfg.Model); err != nil {
			return fmt.Errorf("failed to execute chunk %d: %w", chunk.ChunkNumber, err)
		}

		fmt.Printf("  ✓ Chunk %d/%d completed\n\n", i+1, totalChunks)

		// Log progress
		slog.Info("Chunk executed successfully",
			slog.Int("chunk", chunk.ChunkNumber),
			slog.Int("completed", i+1),
			slog.Int("total", totalChunks),
		)
	}

	return nil
}
