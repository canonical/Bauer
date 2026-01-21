package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"bauer/internal/config"
	"bauer/internal/gdocs"
	"bauer/internal/prompt"
)

func main() {
	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// 2. Setup Logging
	// For now, mirroring POC behavior: logging to log.json in the current directory
	// TODO disable with a flag or env var
	logFile, err := os.OpenFile("log.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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
	fmt.Printf("Extraction complete. Output written to %s\n", outputFile)

	// 6. Initialize Prompt Engine
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
	fmt.Printf("\nGenerating prompts for %d locations (chunk size: %d)...\n", totalLocations, cfg.ChunkSize)

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
	fmt.Printf("\n✓ Generated %d prompt file(s) in '%s/':\n\n", len(chunks), cfg.OutputDir)
	for _, chunk := range chunks {
		fmt.Printf("  [Chunk %d] %s (%d locations)\n",
			chunk.ChunkNumber,
			chunk.Filename,
			chunk.LocationCount,
		)
		slog.Info("Generated chunk",
			slog.Int("chunk_number", chunk.ChunkNumber),
			slog.String("filename", chunk.Filename),
			slog.Int("location_count", chunk.LocationCount),
		)
	}

	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Review the generated prompt files in '%s/'\n", cfg.OutputDir)
	fmt.Printf("  2. Each file is ready to be passed to GitHub Copilot\n")

	if cfg.DryRun {
		fmt.Println("\nDry run enabled: Skipping Copilot execution and PR creation.")
	} else {
		fmt.Println("\nNote: Copilot SDK integration will be implemented in Phase 3")
	}
}
