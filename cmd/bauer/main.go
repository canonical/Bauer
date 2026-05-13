package main

import (
	"bauer/internal/artifacts"
	"bauer/internal/copilotcli"
	"bauer/internal/github"
	"bauer/internal/orchestrator"
	"bauer/internal/source"
	"bauer/internal/workflow"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Parse CLI flags
	githubRepo := flag.String("github-repo", "", "GitHub repository (owner/repo or HTTPS URL)")
	docID := flag.String("doc-id", "", "Google Doc ID")
	credentialsPath := flag.String("credentials", "bau-test-creds.json", "Path to service account credentials JSON")
	localRepoPath := flag.String("local-repo-path", "/tmp/ubuntu.com", "Local path for cloned repository")
	dryRun := flag.Bool("dry-run", false, "Perform a dry run without creating PR")
	outputDir := flag.String("output-dir", "bauer-output", "Output directory for Bauer results")
	branchPrefix := flag.String("branch-prefix", "bauer", "Branch naming prefix")
	artifactsDir := flag.String("artifacts-dir", "", "Directory for append-only run artifacts (default: ./bauer-artifacts)")

	flag.Parse()

	// Validate required flags
	if *githubRepo == "" {
		fmt.Fprintf(os.Stderr, "ERROR: --github-repo is required\n")
		os.Exit(1)
	}
	if *docID == "" {
		fmt.Fprintf(os.Stderr, "ERROR: --doc-id is required\n")
		os.Exit(1)
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Bauer - A tool to automate BAU tasks")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Create workflow input from CLI flags/config
	ghToken, err := github.GetGitHubToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Could not get GitHub token: %v\n", err)
		ghToken = ""
	}

	workflowInput := workflow.WorkflowInput{
		GitHubRepo:    *githubRepo,
		GitHubToken:   ghToken,
		BranchPrefix:  *branchPrefix,
		DocID:         *docID,
		Credentials:   *credentialsPath,
		LocalRepoPath: *localRepoPath,
		DryRun:        *dryRun,
		OutputDir:     *outputDir,
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	copilotClient, err := copilotcli.NewClient(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to create Copilot client: %v\n", err)
		os.Exit(1)
	}

	gdocsAdapter := source.NewGDocsAdapter()
	sources := source.NewManager(gdocsAdapter)

	// Resolve artifacts dir: flag → BAUER_ARTIFACTS_DIR env → default
	resolvedArtifactsDir := *artifactsDir
	if resolvedArtifactsDir == "" {
		resolvedArtifactsDir = os.Getenv("BAUER_ARTIFACTS_DIR")
	}
	if resolvedArtifactsDir == "" {
		resolvedArtifactsDir = "bauer-artifacts"
	}
	artMgr := artifacts.NewManager(resolvedArtifactsDir)

	orch := orchestrator.NewOrchestrator(copilotClient, sources, artMgr)

	// Execute the complete workflow
	result, err := workflow.ExecuteWorkflow(context.Background(), workflowInput, orch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Branch: %s\n", result.RepositoryInfo.BranchName)
	fmt.Printf("PR: %s\n", result.FinalizationInfo.PullRequest.URL)
}
