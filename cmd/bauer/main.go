package main

import (
	"bauer/internal/config"
	"bauer/internal/github"
	"bauer/internal/orchestrator"
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func runGithub() {
	// 1. Parse a GitHub repo (works with all 3 formats)
	// Using default repo for now
	defaultRepo := "canonical/canonical.com"
	repo, err := github.ParseGitHubRepo(defaultRepo)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Parsed: %s/%s\n", repo.Owner, repo.Name)

	// 2. Check if gh CLI is installed
	if !github.IsGhCLIInstalled() {
		log.Fatal("❌ gh CLI not installed.")
	}
	fmt.Println("✅ gh CLI detected")

	// 3. Validate GitHub authentication
	if err := github.ValidateGitHubAuth(); err != nil {
		log.Fatal("❌ Not authenticated. Run: gh auth login")
	}
	fmt.Println("✅ GitHub authenticated")

	// 4. Clone/update repo (creates temp directory)
	localPath := "/tmp/test-bauer-repo"
	if err := github.CloneOrUpdateRepo(repo, localPath); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Repo ready at: %s\n", localPath)

	// 5. Create feature branch
	// Using default branch name for now
	branchName := "bauer/test-phase1"
	if err := github.CreateFeatureBranch(localPath, branchName); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Branch created: %s\n", branchName)

	// 6. Check current branch
	current, err := github.GetCurrentBranch(localPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("✅ Currently on: %s\n", current)
}

func runBauer() error {

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Bauer - A tool to automate BAU tasks")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// 1. Load Configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading configuration\n%w", err)
	}

	// 1b. Change to target repository if specified
	if cfg.TargetRepo != "" {
		// Convert output directory to absolute before changing directory
		absOutputDir, err := filepath.Abs(cfg.OutputDir)
		if err != nil {
			return fmt.Errorf("failed to resolve output directory path: %w", err)
		}
		cfg.OutputDir = absOutputDir

		if err := os.Chdir(cfg.TargetRepo); err != nil {
			return fmt.Errorf("failed to change to target repository %q: %w", cfg.TargetRepo, err)
		}
		cwd, _ := os.Getwd()
		fmt.Printf("Working directory: %s\n\n", cwd)
	}

	// 2. Setup Logging
	// For now, mirroring POC behavior: logging to log.json in the current directory
	// TODO disable with a flag or env var
	logFile, err := os.OpenFile("bauer-log.json", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
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

	// 3. Create and execute orchestrator
	orch := orchestrator.NewOrchestrator()
	result, err := orch.Execute(ctx, cfg)
	if err != nil {
		slog.Error("Orchestration failed", slog.String("error", err.Error()))
		// Check if the error is credentials-related and provide more context
		errMsg := err.Error()
		if strings.Contains(errMsg, "credentials") || strings.Contains(errMsg, "GOOGLE_") {
			fmt.Fprintf(os.Stderr, "\n⚠️  CREDENTIALS ERROR:\n")
			fmt.Fprintf(os.Stderr, "  %v\n\n", err)
			fmt.Fprintf(os.Stderr, "Please verify:\n")
			fmt.Fprintf(os.Stderr, "  1. GOOGLE_* service account variables are set (for example in .env)\n")
			fmt.Fprintf(os.Stderr, "  2. GOOGLE_PRIVATE_KEY contains a valid key (use \\n escapes in .env)\n")
			fmt.Fprintf(os.Stderr, "  3. Required variables include:\n")
			fmt.Fprintf(os.Stderr, "     - GOOGLE_TYPE\n")
			fmt.Fprintf(os.Stderr, "     - GOOGLE_PROJECT_ID\n")
			fmt.Fprintf(os.Stderr, "     - GOOGLE_PRIVATE_KEY\n")
			fmt.Fprintf(os.Stderr, "     - GOOGLE_CLIENT_EMAIL\n")
			fmt.Fprintf(os.Stderr, "     - GOOGLE_TOKEN_URI\n\n")
		}
		return err
	}

	// 4. Print results
	outputFile := "bauer-doc-suggestions.json"
	fmt.Println("[1/4] Extracting from Google Doc...")
	fmt.Printf("  ✓ Extraction completed in %s\n", result.ExtractionDuration.Round(time.Millisecond))
	fmt.Println()

	fmt.Println("[2/4] Generating technical plan...")
	fmt.Printf("  ✓ Saved: %s\n", outputFile)
	fmt.Printf("  ✓ Generated %d chunk file(s) in '%s/'\n", len(result.Chunks), cfg.OutputDir)
	fmt.Printf("  ✓ Planning completed in %s\n", result.PlanDuration.Round(time.Millisecond))
	for _, chunk := range result.Chunks {
		slog.Info("Generated chunk",
			slog.Int("chunk_number", chunk.ChunkNumber),
			slog.String("filename", chunk.Filename),
			slog.Int("location_count", chunk.LocationCount),
		)
	}
	fmt.Println()

	if result.DryRun {
		fmt.Println("[3/4] Copilot execution (skipped - dry run)")
		fmt.Println()
		fmt.Println(strings.Repeat("=", 80))
		fmt.Println("DRY RUN COMPLETE")
		fmt.Println(strings.Repeat("=", 80))
		fmt.Printf("  Summary:\n")
		fmt.Printf("    • Extracted: %d suggestions\n", len(result.ExtractionResult.ActionableSuggestions))
		fmt.Printf("    • Grouped into: %d location(s)\n", len(result.ExtractionResult.GroupedSuggestions))
		fmt.Printf("    • Generated: %d chunk file(s) in '%s/'\n", len(result.Chunks), cfg.OutputDir)
		fmt.Printf("\n  Timing:\n")
		fmt.Printf("    • Extraction: %s\n", result.ExtractionDuration.Round(time.Millisecond))
		fmt.Printf("    • Planning: %s\n", result.PlanDuration.Round(time.Millisecond))
		fmt.Printf("    • Total: %s\n", result.TotalDuration.Round(time.Millisecond))
		fmt.Printf("\n  Next steps:\n")
		fmt.Printf("    1. Review generated chunks in '%s/'\n", cfg.OutputDir)
		fmt.Printf("    2. Run without --dry-run to execute changes via Copilot\n")
		fmt.Printf("    3. Or manually pass chunk files to: gh copilot\n")
		fmt.Println(strings.Repeat("=", 80))
		return nil
	}

	// 5. Copilot execution results
	fmt.Println("[3/4] Executing changes via Copilot...")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("  ✓ All %d chunk(s) processed successfully\n", len(result.CopilotOutputs))
	fmt.Printf("  ✓ Total Copilot execution time: %s\n", result.CopilotDuration.Round(time.Millisecond))
	fmt.Println()

	// 6. Summary results if multiple chunks
	if len(result.Chunks) > 1 {
		fmt.Println("[4/5] Generating summary...")
		fmt.Printf("  ✓ Summary completed in %s\n", result.SummaryDuration.Round(time.Millisecond))
		fmt.Println()
	}

	// 7. Final summary
	stepLabel := "[4/4]"
	if len(result.Chunks) > 1 {
		stepLabel = "[5/5]"
	}

	fmt.Printf("%s Complete!\n", stepLabel)
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("SUCCESS: Feedback applied!")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Printf("  Summary:\n")
	fmt.Printf("    • Extracted: %d suggestions\n", len(result.ExtractionResult.ActionableSuggestions))
	fmt.Printf("    • Processed: %d chunk(s)\n", len(result.Chunks))
	fmt.Printf("\n  Timing:\n")
	fmt.Printf("    • Extraction: %s\n", result.ExtractionDuration.Round(time.Millisecond))
	fmt.Printf("    • Planning: %s\n", result.PlanDuration.Round(time.Millisecond))
	fmt.Printf("    • Copilot execution: %s\n", result.CopilotDuration.Round(time.Millisecond))
	fmt.Printf("    • Total: %s\n", result.TotalDuration.Round(time.Millisecond))
	fmt.Printf("\n  Next steps:\n")
	fmt.Printf("    • Review the changes made by Copilot\n")
	fmt.Printf("    • Create a PR with: gh pr create\n")
	fmt.Println(strings.Repeat("=", 80))

	return nil
}

func main() {
	runGithub()

	if err := runBauer(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}
}
