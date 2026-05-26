package workflow

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"bauer/internal/config"
	"bauer/internal/github"
	"bauer/internal/orchestrator"
)

// WorkflowInput represents the input for a complete workflow execution
type WorkflowInput struct {
	// GitHub configuration
	GitHubRepo   string
	GitHubToken  string
	BranchPrefix string

	// Bauer configuration
	DocID       string
	Credentials string
	ChunkSize   int
	PageRefresh bool
	OutputDir   string
	Model       string
	DryRun      bool

	// Local repository path
	LocalRepoPath string
}

// WorkflowOutput represents the complete workflow execution result
type WorkflowOutput struct {
	// GitHub Setup
	RepositoryInfo struct {
		Owner         string
		Repo          string
		LocalPath     string
		BranchName    string
		DefaultBranch string
		CurrentBranch string
	} `json:"repository_info"`

	// Bauer Processing
	BauerResult struct {
		ExtractionDuration time.Duration `json:"extraction_duration"`
		PlanDuration       time.Duration `json:"plan_duration"`
		CopilotDuration    time.Duration `json:"copilot_duration"`
		ChunkCount         int           `json:"chunk_count"`
		TotalSuggestions   int           `json:"total_suggestions"`
	} `json:"bauer_result"`

	// GitHub Finalization
	FinalizationInfo struct {
		CommitMessage string
		BranchPushed  bool
		PullRequest   struct {
			URL    string
			Number int
			Title  string
		}
	} `json:"finalization_info"`

	// Overall
	Status        string        `json:"status"` // "success", "partial", "failed"
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	TotalDuration time.Duration `json:"total_duration"`
	Errors        []string      `json:"errors"`
	Warnings      []string      `json:"warnings"`
}

// ExecuteWorkflow orchestrates the complete flow:
// 1. GitHub Setup (clone, create branch)
// 2. Bauer Processing (extract, chunk, apply changes)
// 3. GitHub Finalization (commit, push, create PR)
func ExecuteWorkflow(ctx context.Context, input WorkflowInput, orch orchestrator.Orchestrator) (*WorkflowOutput, error) {
	output := &WorkflowOutput{
		Status:    "pending",
		StartTime: time.Now(),
		Errors:    []string{},
		Warnings:  []string{},
	}

	logger := slog.Default()

	// GitHub setup
	logger.Info("workflow: Setting up GitHub")

	githubSetupInput := github.GitHubSetupInput{
		GitHubRepo:    input.GitHubRepo,
		GitHubToken:   input.GitHubToken,
		BranchPrefix:  input.BranchPrefix,
		LocalRepoPath: input.LocalRepoPath,
	}

	githubSetupOutput, err := github.SetupGitHubPhase(githubSetupInput)
	if err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, err.Error())
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}
	// Store GH setup results
	output.RepositoryInfo.Owner = githubSetupOutput.Repo.Owner
	output.RepositoryInfo.Repo = githubSetupOutput.Repo.Name
	output.RepositoryInfo.LocalPath = githubSetupOutput.LocalPath
	output.RepositoryInfo.BranchName = githubSetupOutput.BranchName
	output.RepositoryInfo.DefaultBranch = githubSetupOutput.DefaultBranch
	output.RepositoryInfo.CurrentBranch = githubSetupOutput.CurrentBranch

	logger.Info("workflow success: GitHub setup successful")

	// Convert credentials path to absolute
	// Do this before changing directory so relative paths work
	var credentialsPath string
	if input.Credentials != "" {
		absPath, err := filepath.Abs(input.Credentials)
		if err != nil {
			output.Status = "failed"
			output.Errors = append(output.Errors, fmt.Sprintf("failed to resolve credentials path: %v", err))
			output.EndTime = time.Now()
			output.TotalDuration = output.EndTime.Sub(output.StartTime)
			return output, err
		}
		credentialsPath = absPath
		logger.Info("workflow: resolved credentials path", "path", credentialsPath)
	}

	// Change to target repository directory
	// Save original directory to restore later
	originalDir, err := os.Getwd()
	if err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to get current directory: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}

	if err := os.Chdir(input.LocalRepoPath); err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to change to cloned repository: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}
	logger.Info("workflow: changed to cloned repository", "path", input.LocalRepoPath)
	defer os.Chdir(originalDir)

	// Bauer processing
	logger.Info("workflow: starting phase 2 - Bauer processing")

	bauerStartTime := time.Now()

	// Create Bauer config with target repo (now current directory)
	bauerCfg := &config.Config{
		DocID:           input.DocID,
		CredentialsPath: credentialsPath, // Use absolute path
		DryRun:          config.BoolPtr(input.DryRun),
		ChunkSize:       input.ChunkSize,
		PageRefresh:     config.BoolPtr(input.PageRefresh),
		OutputDir:       input.OutputDir,
		Model:           input.Model,
		TargetRepo:      ".", // Current directory is the cloned repo
	}

	logger.Info("workflow: Bauer target repository set at", "path", bauerCfg.TargetRepo)

	// Execute Bauer orchestration
	bauerResult, err := orch.Execute(ctx, bauerCfg)
	if err != nil {
		output.Status = "partial"
		output.Errors = append(output.Errors, fmt.Sprintf("Bauer processing error: %v", err))
		logger.Warn("workflow: Bauer processing returned error", "error", err)
		// Continue anyway - we can still commit what we have
	}

	// Store Bauer results
	if bauerResult != nil {
		output.BauerResult.ExtractionDuration = bauerResult.ExtractionDuration
		output.BauerResult.PlanDuration = bauerResult.PlanDuration
		output.BauerResult.CopilotDuration = bauerResult.AgentDuration
		if len(bauerResult.Chunks) > 0 {
			output.BauerResult.ChunkCount = len(bauerResult.Chunks)
		}
		if bauerResult.ExtractionBundle != nil && bauerResult.ExtractionBundle.Document != nil {
			// Count total suggestions from extraction result
			output.BauerResult.TotalSuggestions = 0 // TODO: adjust based on actual field
		}
	}

	logger.Info("Bauer results",
		"extraction_duration", output.BauerResult.ExtractionDuration,
		"plan_duration", output.BauerResult.PlanDuration,
		"copilot_duration", output.BauerResult.CopilotDuration,
		"chunk_count", output.BauerResult.ChunkCount,
		"total_suggestions", output.BauerResult.TotalSuggestions,
	)
	output.BauerResult.CopilotDuration = time.Since(bauerStartTime)
	logger.Info("workflow success: Bauer processing finished")

	// GitHub finalization
	logger.Info("workflow: GitHub finalization")

	commitMessage := fmt.Sprintf("Apply BAU suggestions from doc %s", input.DocID)
	prTitle := fmt.Sprintf("Apply BAU suggestions to %s", githubSetupOutput.Repo.Name)
	prBody := fmt.Sprintf("Automated copy update changes from Bauer\n\nGDoc ID: %s", input.DocID)

	finalizationInput := github.GitHubFinalizationInput{
		LocalRepoPath: input.LocalRepoPath,
		BranchName:    githubSetupOutput.BranchName,
		DefaultBranch: githubSetupOutput.DefaultBranch,
		Owner:         githubSetupOutput.Repo.Owner,
		Repo:          githubSetupOutput.Repo.Name,
		CommitMessage: commitMessage,
		DryRun:        input.DryRun,
		PRTitle:       prTitle,
		PRBody:        prBody,
		Labels:        []string{},
	}

	finalizationOutput, _ := github.FinalizeGitHubPhase(finalizationInput)

	// Store GH PR results
	output.FinalizationInfo.CommitMessage = finalizationOutput.CommitMessage
	output.FinalizationInfo.BranchPushed = finalizationOutput.BranchPushed
	output.FinalizationInfo.PullRequest.URL = finalizationOutput.PullRequest.URL
	output.FinalizationInfo.PullRequest.Title = finalizationOutput.PullRequest.Title

	// Merge warnings and errors from finalization
	output.Warnings = append(output.Warnings, finalizationOutput.Warnings...)
	output.Errors = append(output.Errors, finalizationOutput.Errors...)

	logger.Info("workflow: phase 3 complete - GitHub finalization finished")

	output.EndTime = time.Now()
	output.TotalDuration = output.EndTime.Sub(output.StartTime)

	if len(output.Errors) == 0 {
		output.Status = "success"
	} else if output.FinalizationInfo.BranchPushed {
		output.Status = "partial"
	} else {
		output.Status = "failed"
	}

	logger.Info("workflow: complete",
		"status", output.Status,
		"duration", output.TotalDuration,
		"errors", len(output.Errors),
		"warnings", len(output.Warnings),
	)

	return output, nil
}
