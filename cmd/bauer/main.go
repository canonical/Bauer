package main

import (
	"bauer/internal/config"
	"bauer/internal/copilotcli"
	"bauer/internal/github"
	"bauer/internal/orchestrator"
	"bauer/internal/source"
	"bauer/internal/workflow"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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

	// Resolve credentials path to absolute so it remains valid after directory changes.
	absCredentials, err := filepath.Abs(*credentialsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to resolve credentials path: %v\n", err)
		os.Exit(1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to get working directory: %v\n", err)
		os.Exit(1)
	}

	copilotAgent, err := copilotcli.NewClient(cwd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: failed to create Copilot client: %v\n", err)
		os.Exit(1)
	}

	sources := source.NewManager(absCredentials)
	orch := orchestrator.New(copilotAgent, sources)

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

// resolveCLIConfig builds a Config from CLI flags, falling back to environment variables
// when flag values are empty. It does NOT call Validate — callers must do that separately.
func resolveCLIConfig(flags config.CLIFlags) (*config.Config, error) {
	docID := flags.DocID
	if docID == "" {
		docID = os.Getenv("BAUER_DOC_ID")
	}

	credentialsPath := flags.CredentialsPath
	if credentialsPath == "" {
		credentialsPath = os.Getenv("BAUER_CREDENTIALS")
	}

	outputDir := flags.OutputDir
	if outputDir == "" {
		outputDir = os.Getenv("BAUER_OUTPUT_DIR")
	}

	model := flags.Model
	if model == "" {
		model = os.Getenv("BAUER_MODEL")
	}

	summaryModel := flags.SummaryModel
	if summaryModel == "" {
		summaryModel = os.Getenv("BAUER_SUMMARY_MODEL")
	}

	targetRepo := flags.TargetRepo
	if targetRepo == "" {
		targetRepo = os.Getenv("BAUER_TARGET_REPO")
	}

	return &config.Config{
		DocID:           docID,
		CredentialsPath: credentialsPath,
		DryRun:          flags.DryRun,
		ChunkSize:       flags.ChunkSize,
		OutputDir:       outputDir,
		Model:           model,
		SummaryModel:    summaryModel,
		TargetRepo:      targetRepo,
	}, nil
}

// openPRExecutionConfig returns a copy of cfg with DryRun forced to false,
// so that the Copilot agent runs even when the overall --dry-run flag was set.
func openPRExecutionConfig(cfg *config.Config) *config.Config {
	clone := *cfg
	f := false
	clone.DryRun = &f
	return &clone
}
