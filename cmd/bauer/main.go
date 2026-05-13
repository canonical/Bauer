package main

import (
	"bauer/internal/artifacts"
	"bauer/internal/config"
	"bauer/internal/copilotcli"
	"bauer/internal/github"
	"bauer/internal/orchestrator"
	"bauer/internal/source"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	ctx := context.Background()

	// 1. Parse flags
	flags, err := config.ParseCLIFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 2. Mutual exclusion check — before any network calls
	if config.BoolVal(flags.OpenPR, false) && config.BoolVal(flags.OpenIssue, false) {
		fmt.Fprintln(os.Stderr, "Error: --open-pr and --open-issue are mutually exclusive. Pick one.")
		fmt.Fprintln(os.Stderr, "  Use --open-pr to apply changes and open a PR.")
		fmt.Fprintln(os.Stderr, "  Use --open-issue to generate a plan and open an issue without applying changes.")
		os.Exit(1)
	}

	// 3. Resolve config
	cfg, err := config.NewResolver(
		config.NewEnvVarSource(),
		config.NewFlagsSource(flags),
		config.NewDefaultsSource(),
	).Resolve()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// 4. Figma token validation (fail fast)
	if cfg.FigmaURL != "" && cfg.FigmaToken == "" {
		fmt.Fprintln(os.Stderr, "Error: BAUER_FIGMA_TOKEN or FIGMA_TOKEN must be set when --figma-url is supplied.")
		os.Exit(1)
	}

	// Resolve output and artifacts dirs to absolute paths before any chdir
	absOutputDir, err := filepath.Abs(cfg.OutputDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: resolve output dir: %v\n", err)
		os.Exit(1)
	}
	cfg.OutputDir = absOutputDir

	absArtifactsDir, err := filepath.Abs(cfg.ArtifactsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: resolve artifacts dir: %v\n", err)
		os.Exit(1)
	}
	cfg.ArtifactsDir = absArtifactsDir

	// 5. Setup orchestrator
	artMgr := artifacts.NewManager(absArtifactsDir)
	gdocsAdapter := source.NewGDocsAdapter()
	sources := source.NewManager(gdocsAdapter)

	newAgent := func() (orchestrator.Agent, error) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		return copilotcli.NewClient(cwd)
	}

	// 6. Dispatch to mode
	switch {
	case config.BoolVal(cfg.OpenIssue, false):
		if err := runOpenIssue(ctx, cfg, sources, artMgr); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case config.BoolVal(cfg.OpenPR, false):
		if err := runOpenPR(ctx, cfg, sources, artMgr, newAgent); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		if err := runStandalone(ctx, cfg, sources, artMgr, newAgent); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func runStandalone(ctx context.Context, cfg *config.Config, sources *source.Manager, artMgr *artifacts.Manager, newAgent func() (orchestrator.Agent, error)) error {
	agent, err := newAgent()
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	orch := orchestrator.NewOrchestrator(agent, sources, artMgr)
	result, err := orch.Execute(ctx, cfg)
	if err != nil {
		return err
	}

	fmt.Printf("Status: %s\n", resultStatus(result))
	return nil
}

func runOpenPR(ctx context.Context, cfg *config.Config, sources *source.Manager, artMgr *artifacts.Manager, newAgent func() (orchestrator.Agent, error)) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Read remote from git config
	owner, repo, err := github.ReadRemoteFromGitConfig(ctx, cwd)
	if err != nil {
		return fmt.Errorf("--open-pr requires a git repo with an 'origin' remote: %w", err)
	}

	// Check GitHub token before running Copilot
	if _, err := github.GetGitHubToken(); err != nil {
		return fmt.Errorf("--open-pr requires GitHub auth: %w", err)
	}

	// Run Copilot (always; we want to see the changes)
	agent, err := newAgent()
	if err != nil {
		return fmt.Errorf("create agent: %w", err)
	}

	orch := orchestrator.NewOrchestrator(agent, sources, artMgr)
	_, err = orch.Execute(ctx, cfg)
	if err != nil {
		return fmt.Errorf("orchestration failed: %w", err)
	}

	// If dry-run, skip PR creation
	if config.BoolVal(cfg.DryRun, false) {
		fmt.Println("dry-run: changes applied locally, PR creation skipped")
		return nil
	}

	// Create branch from main, commit, push
	branchName := fmt.Sprintf("%s/doc-suggestions-%d", cfg.BranchPrefix, time.Now().Unix())
	if err := github.CreateAndPushBranch(ctx, cwd, branchName); err != nil {
		return fmt.Errorf("failed to create/push branch: %w", err)
	}

	// Commit changes
	commitMsg := fmt.Sprintf("Apply BAU suggestions from doc %s", cfg.DocID)
	if err := github.CommitChanges(cwd, commitMsg); err != nil {
		// "no changes to commit" is acceptable if Copilot made no modifications
		if !strings.Contains(err.Error(), "no changes to commit") {
			return fmt.Errorf("failed to commit changes: %w", err)
		}
		fmt.Println("warning: no changes to commit")
	}

	// Push branch
	if err := github.PushBranch(cwd, branchName); err != nil {
		return fmt.Errorf("failed to push branch: %w", err)
	}

	// Create PR
	prTitle := fmt.Sprintf("Apply BAU suggestions from doc %s", cfg.DocID)
	prURL, err := github.CreatePR(owner, repo, github.CreatePROptions{
		Title:      prTitle,
		HeadBranch: branchName,
		BaseBranch: "main",
	})
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	fmt.Printf("PR created: %s\n", prURL)
	return nil
}

func runOpenIssue(ctx context.Context, cfg *config.Config, sources *source.Manager, artMgr *artifacts.Manager) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	// Read remote from git config
	owner, repo, err := github.ReadRemoteFromGitConfig(ctx, cwd)
	if err != nil {
		return fmt.Errorf("--open-issue requires a git repo with an 'origin' remote: %w", err)
	}

	// Check GitHub token before any API calls
	if _, err := github.GetGitHubToken(); err != nil {
		return fmt.Errorf("--open-issue requires GitHub auth: %w", err)
	}

	// Run orchestrator in dry-run mode (extraction + chunking, no Copilot)
	dryRunCfg := *cfg
	dryRunCfg.DryRun = config.BoolPtr(true)

	orch := orchestrator.NewOrchestrator(nil, sources, artMgr)
	result, err := orch.Execute(ctx, &dryRunCfg)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	title := fmt.Sprintf("BAU: Apply suggestions from doc %s", cfg.DocID)
	body := formatIssueBody(result, cfg.DocID)

	issueURL, issueNum, err := github.CreateIssue(ctx, owner, repo, title, body)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	fmt.Printf("Issue #%d created: %s\n", issueNum, issueURL)
	return nil
}

func formatIssueBody(result *orchestrator.OrchestrationResult, docID string) string {
	var b strings.Builder
	b.WriteString("## BAU Suggestions from Google Doc\n\n")
	b.WriteString(fmt.Sprintf("**Doc ID**: `%s`\n", docID))

	if result != nil && result.Bundle != nil && result.Bundle.Document != nil {
		doc := result.Bundle.Document
		totalSuggestions := len(doc.GroupedSuggestions)
		chunkCount := len(result.Chunks)
		b.WriteString(fmt.Sprintf("**Total suggestions**: %d across %d chunk(s)\n\n", totalSuggestions, chunkCount))

		for _, chunk := range result.Chunks {
			b.WriteString(fmt.Sprintf("### Chunk %d\n\n", chunk.ChunkNumber))
			b.WriteString(fmt.Sprintf("- **File**: %s\n", chunk.Filename))
			b.WriteString(fmt.Sprintf("- **Locations**: %d\n\n", chunk.LocationCount))
		}
	} else {
		b.WriteString("No suggestions extracted.\n")
	}

	return b.String()
}

func resultStatus(result *orchestrator.OrchestrationResult) string {
	if result == nil {
		return "failed"
	}
	if result.DryRun {
		return "dry-run"
	}
	return "success"
}
