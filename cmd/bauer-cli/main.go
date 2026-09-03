package main

import (
	"bauer/internal/gdocs"
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
	githubRepo := flag.String("github-repo", "", "GitHub repository (owner/repo or HTTPS URL)")
	docID := flag.String("doc-id", "", "Google Doc ID")
	localRepoPath := flag.String("local-repo-path", "/tmp/ubuntu.com", "Local path for cloned repository")
	parseOnly := flag.Bool("parse-only", false, "Parse document and output machine-readable JSON only")
	outputDir := flag.String("output-dir", "bauer-output", "Output directory for Bauer results")
	branchPrefix := flag.String("branch-prefix", "bauer", "Branch naming prefix")

	flag.Parse()

	// Validate required flags
	if *docID == "" {
		fmt.Fprintf(os.Stderr, "ERROR: --doc-id is required\n")
		os.Exit(1)
	}

	// GitHub repo is optional for parse-only mode
	if !*parseOnly && *githubRepo == "" {
		fmt.Fprintf(os.Stderr, "ERROR: --github-repo is required (unless using --parse-only)\n")
		os.Exit(1)
	}

	// Credentials are assembled from environment variables (APP_PROJECT_ID,
	// APP_PRIVATE_KEY, APP_CLIENT_EMAIL, APP_CLIENT_ID).
	credentialsJSON, err := gdocs.CredentialsFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(strings.Repeat("=", 80))
	fmt.Println("Bauer - A tool to automate BAU tasks")
	fmt.Println(strings.Repeat("=", 80))
	fmt.Println()

	// Create workflow input from CLI flags/config
	ghToken := ""
	if *githubRepo != "" {
		var err error
		ghToken, err = github.GetGitHubToken()
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Could not get GitHub token: %v\n", err)
		}
	}

	workflowInput := workflow.WorkflowInput{
		GitHubRepo:      *githubRepo,
		GitHubToken:     ghToken,
		BranchPrefix:    *branchPrefix,
		DocID:           *docID,
		CredentialsJSON: credentialsJSON,
		LocalRepoPath:   *localRepoPath,
		ParseOnly:       *parseOnly,
		OutputDir:       *outputDir,
	}

	orch := orchestrator.NewOrchestrator()

	// Execute the complete workflow
	result, err := workflow.ExecuteWorkflow(context.Background(), workflowInput, orch)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	// Print results
	if *parseOnly {
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Output file: %s\n", result.OutputFile)
	} else {
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Output file: %s\n", result.OutputFile)
		fmt.Printf("Branch: %s\n", result.RepositoryInfo.BranchName)
		fmt.Printf("Issue: %s\n", result.FinalizationInfo.Issue.URL)
	}
}
