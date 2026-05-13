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
	"strconv"
	"strings"
)

// envFallback returns the env var value if the flag value is empty/zero.
func envFallback(flagVal, envKey string) string {
	if flagVal != "" {
		return flagVal
	}
	return os.Getenv(envKey)
}

// envBoolFallback returns the env var bool if the flag value is false.
func envBoolFallback(flagVal bool, envKey string) bool {
	if flagVal {
		return flagVal
	}
	v := os.Getenv(envKey)
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: invalid bool for %s=%q: %v\n", envKey, v, err)
		return false
	}
	return b
}

// envIntFallback returns the env var int if the flag value is zero.
func envIntFallback(flagVal int, envKey string) int {
	if flagVal != 0 {
		return flagVal
	}
	v := os.Getenv(envKey)
	if v == "" {
		return 0
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: invalid int for %s=%q: %v\n", envKey, v, err)
		return 0
	}
	return i
}

func main() {
	// Parse CLI flags
	githubRepo := flag.String("github-repo", "", "GitHub repository (owner/repo or HTTPS URL)")
	docID := flag.String("doc-id", "", "Google Doc ID")
	credentialsPath := flag.String("credentials", "", "Path to service account credentials JSON")
	localRepoPath := flag.String("local-repo-path", "/tmp/ubuntu.com", "Local path for cloned repository")
	dryRun := flag.Bool("dry-run", false, "Perform a dry run without creating PR")
	outputDir := flag.String("output-dir", "", "Output directory for Bauer results")
	branchPrefix := flag.String("branch-prefix", "", "Branch naming prefix")
	artifactsDir := flag.String("artifacts-dir", "", "Directory for append-only run artifacts (default: ./bauer-artifacts)")
	chunkSize := flag.Int("chunk-size", 0, "Total number of chunks to create")
	model := flag.String("model", "", "Copilot model to use for sessions")

	flag.Parse()

	// Resolve with env var fallbacks: flag → BAUER_* env → default
	resolvedGithubRepo := envFallback(*githubRepo, "BAUER_GITHUB_REPO")
	resolvedDocID := envFallback(*docID, "BAUER_DOC_ID")
	resolvedCredentials := envFallback(*credentialsPath, "BAUER_CREDENTIALS_PATH")
	if resolvedCredentials == "" {
		resolvedCredentials = "bau-test-creds.json"
	}
	resolvedOutputDir := envFallback(*outputDir, "BAUER_OUTPUT_DIR")
	if resolvedOutputDir == "" {
		resolvedOutputDir = "bauer-output"
	}
	resolvedBranchPrefix := envFallback(*branchPrefix, "BAUER_BRANCH_PREFIX")
	if resolvedBranchPrefix == "" {
		resolvedBranchPrefix = "bauer"
	}
	resolvedArtifactsDir := envFallback(*artifactsDir, "BAUER_ARTIFACTS_DIR")
	if resolvedArtifactsDir == "" {
		resolvedArtifactsDir = "bauer-artifacts"
	}
	resolvedModel := envFallback(*model, "BAUER_MODEL")
	if resolvedModel == "" {
		resolvedModel = "gpt-5-mini-high"
	}
	resolvedChunkSize := envIntFallback(*chunkSize, "BAUER_CHUNK_SIZE")
	resolvedDryRun := envBoolFallback(*dryRun, "BAUER_DRY_RUN")

	// Validate required flags
	if resolvedGithubRepo == "" {
		fmt.Fprintf(os.Stderr, "ERROR: --github-repo or BAUER_GITHUB_REPO is required\n")
		os.Exit(1)
	}
	if resolvedDocID == "" {
		fmt.Fprintf(os.Stderr, "ERROR: --doc-id or BAUER_DOC_ID is required\n")
		os.Exit(1)
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Bauer - A tool to automate BAU tasks")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Create workflow input from resolved config
	ghToken, err := github.GetGitHubToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Could not get GitHub token: %v\n", err)
		ghToken = ""
	}

	workflowInput := workflow.WorkflowInput{
		GitHubRepo:    resolvedGithubRepo,
		GitHubToken:   ghToken,
		BranchPrefix:  resolvedBranchPrefix,
		DocID:         resolvedDocID,
		Credentials:   resolvedCredentials,
		LocalRepoPath: *localRepoPath,
		DryRun:        resolvedDryRun,
		OutputDir:     resolvedOutputDir,
		ChunkSize:     resolvedChunkSize,
		Model:         resolvedModel,
	}

	artMgr := artifacts.NewManager(resolvedArtifactsDir)
	gdocsAdapter := source.NewGDocsAdapter()
	sources := source.NewManager(gdocsAdapter)

	// Copilot client factory — called after chdir so cwd is the target repo
	newAgent := func() (orchestrator.Agent, error) {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}
		return copilotcli.NewClient(cwd)
	}

	orch := orchestrator.NewOrchestrator(nil, sources, artMgr)

	// Execute the complete workflow
	result, err := workflow.ExecuteWorkflow(context.Background(), workflowInput, orch, newAgent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Print results
	fmt.Printf("Status: %s\n", result.Status)
	fmt.Printf("Branch: %s\n", result.RepositoryInfo.BranchName)
	fmt.Printf("PR: %s\n", result.FinalizationInfo.PullRequest.URL)
}
