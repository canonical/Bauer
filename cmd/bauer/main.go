package main

import (
	"bauer/internal/config"
	"bauer/internal/github"
	"bauer/internal/orchestrator"
	"bauer/internal/workflow"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	// Parse CLI flags
	configFile := flag.String("config", "config.json", "Path to JSON config file")
	githubRepo := flag.String("github-repo", "", "GitHub repository (owner/repo or HTTPS URL)")
	docID := flag.String("doc-id", "", "Google Doc ID")
	credentialsPath := flag.String("credentials", "bau-test-creds.json", "Path to service account credentials JSON")
	localRepoPath := flag.String("local-repo-path", "/tmp/ubuntu.com", "Local path for cloned repository")
	dryRun := flag.Bool("dry-run", false, "Perform a dry run without creating PR")
	outputDir := flag.String("output-dir", "bauer-output", "Output directory for Bauer results")
	branchPrefix := flag.String("branch-prefix", "bauer", "Branch naming prefix")

	flag.Parse()

	// Load from config file if provided, otherwise use command-line flags
	var cfg *config.Config
	var err error

	if *configFile != "" {
		cfg, err = config.LoadFromJSONFile(*configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: Failed to load config file: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Validate required flags
		if *githubRepo == "" {
			fmt.Fprintf(os.Stderr, "ERROR: --github-repo is required\n")
			os.Exit(1)
		}
		if *docID == "" {
			fmt.Fprintf(os.Stderr, "ERROR: --doc-id is required\n")
			os.Exit(1)
		}
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Bauer - A tool to automate BAU tasks")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Prepare workflow input from config
	ghToken, err := github.GetGitHubToken()
	if err != nil {
		fmt.Fprintf(os.Stderr, "WARNING: Could not get GitHub token: %v\n", err)
		ghToken = ""
	}

	// Determine values to use (CLI flags override config file values)
	repoToUse := *githubRepo
	if repoToUse == "" && cfg != nil {
		repoToUse = cfg.GitHubRepo
	}

	docToUse := *docID
	if docToUse == "" && cfg != nil {
		docToUse = cfg.DocID
	}

	credsToUse := *credentialsPath
	if cfg != nil && cfg.CredentialsPath != "" {
		credsToUse = cfg.CredentialsPath
	}

	localRepoToUse := *localRepoPath
	if cfg != nil && cfg.LocalRepoPath != "" {
		localRepoToUse = cfg.LocalRepoPath
	}

	dryRunToUse := *dryRun
	if cfg != nil {
		dryRunToUse = cfg.DryRun
	}

	outputDirToUse := *outputDir
	if cfg != nil && cfg.OutputDir != "" {
		outputDirToUse = cfg.OutputDir
	}

	branchPrefixToUse := *branchPrefix
	if cfg != nil && cfg.BranchPrefix != "" {
		branchPrefixToUse = cfg.BranchPrefix
	}

	workflowInput := workflow.WorkflowInput{
		GitHubRepo:    repoToUse,
		GitHubToken:   ghToken,
		BranchPrefix:  branchPrefixToUse,
		DocID:         docToUse,
		Credentials:   credsToUse,
		LocalRepoPath: localRepoToUse,
		DryRun:        dryRunToUse,
		OutputDir:     outputDirToUse,
	}

	orch := orchestrator.NewOrchestrator()

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
