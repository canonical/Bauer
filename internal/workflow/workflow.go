package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"bauer/internal/config"
	"bauer/internal/github"
	"bauer/internal/orchestrator"
	"bauer/internal/prompt"
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
	ParseOnly   bool // Parse document to JSON only.

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
		ChunkCount         int           `json:"chunk_count"`
		TotalSuggestions   int           `json:"total_suggestions"`
	} `json:"bauer_result"`

	// GitHub Finalization
	FinalizationInfo struct {
		CommitMessage string
		BranchPushed  bool
		Issue         struct {
			URL   string
			Title string
		}
		PullRequest struct {
			URL    string
			Number int
			Title  string
		}
	} `json:"finalization_info"`

	// Parse-only output
	OutputFile string `json:"output_file,omitempty"` // Path to generated JSON for parse-only mode

	// Overall
	Status        string        `json:"status"` // "success", "partial", "failed"
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
	TotalDuration time.Duration `json:"total_duration"`
	Errors        []string      `json:"errors"`
	Warnings      []string      `json:"warnings"`
}

// ExecuteWorkflow orchestrates a parse-first flow:
// 1. Parse Google Doc and write bauer-parse-result.json
// 2. If ParseOnly is false, push the JSON to a branch and create a Copilot issue
func ExecuteWorkflow(ctx context.Context, input WorkflowInput, orch orchestrator.Orchestrator) (*WorkflowOutput, error) {
	output := &WorkflowOutput{
		Status:    "pending",
		StartTime: time.Now(),
		Errors:    []string{},
		Warnings:  []string{},
	}

	logger := slog.Default()

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

	logger.Info("workflow: parse phase")

	bauerCfg := &config.Config{
		DocID:           input.DocID,
		CredentialsPath: credentialsPath,
		ChunkSize:       input.ChunkSize,
		PageRefresh:     input.PageRefresh,
		OutputDir:       input.OutputDir,
		TargetRepo:      ".",
		ParseOnly:       true,
	}

	bauerCfg.ApplyDefaults()
	input.OutputDir = bauerCfg.OutputDir

	bauerResult, err := orch.Execute(ctx, bauerCfg)
	if err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("Bauer parsing error: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}

	if bauerResult != nil {
		output.BauerResult.ExtractionDuration = bauerResult.ExtractionDuration
		output.BauerResult.PlanDuration = 0
		if bauerResult.ParseResult != nil {
			output.BauerResult.TotalSuggestions = len(bauerResult.ParseResult.ActionableSuggestions)
		}
	}

	absOutputDir, err := filepath.Abs(input.OutputDir)
	if err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to resolve output directory path: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}

	if err := os.MkdirAll(absOutputDir, 0755); err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to create output directory: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}

	outputPath := filepath.Join(absOutputDir, "bauer-parse-result.json")
	if bauerResult != nil && bauerResult.ParseResult != nil {
		parseResultJSON, err := json.MarshalIndent(bauerResult.ParseResult, "", "  ")
		if err != nil {
			output.Status = "failed"
			output.Errors = append(output.Errors, fmt.Sprintf("failed to marshal parse result: %v", err))
			output.EndTime = time.Now()
			output.TotalDuration = output.EndTime.Sub(output.StartTime)
			return output, err
		}

		if err := os.WriteFile(outputPath, parseResultJSON, 0644); err != nil {
			output.Status = "failed"
			output.Errors = append(output.Errors, fmt.Sprintf("failed to write parse result file: %v", err))
			output.EndTime = time.Now()
			output.TotalDuration = output.EndTime.Sub(output.StartTime)
			return output, err
		}
	}

	if _, err := os.Stat(outputPath); err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("parse result file was not created at expected location: %s: %v", outputPath, err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}

	output.OutputFile = outputPath

	if input.ParseOnly {
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		output.Status = "success"
		logger.Info("workflow: parse-only mode complete", "output_file", output.OutputFile)
		return output, nil
	}

	repo, err := github.ParseGitHubRepo(input.GitHubRepo)
	if err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("invalid github repo: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}

	token := input.GitHubToken
	if token == "" {
		token, err = github.GetGitHubToken()
		if err != nil {
			output.Status = "failed"
			output.Errors = append(output.Errors, fmt.Sprintf("failed to get github token: %v", err))
			output.EndTime = time.Now()
			output.TotalDuration = output.EndTime.Sub(output.StartTime)
			return output, err
		}
	}

	if err := github.SetupGitHubAuth(token); err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to setup github auth: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}

	parseFileContent, err := os.ReadFile(outputPath)
	if err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to read parse result for issue: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}

	setupInput := github.GitHubSetupInput{
		GitHubRepo:    input.GitHubRepo,
		GitHubToken:   token,
		BranchPrefix:  input.BranchPrefix,
		LocalRepoPath: input.LocalRepoPath,
	}
	setupOutput, err := github.SetupGitHubPhase(setupInput)
	if err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to prepare branch-backed prompt file: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}

	output.RepositoryInfo.Owner = setupOutput.Repo.Owner
	output.RepositoryInfo.Repo = setupOutput.Repo.Name
	output.RepositoryInfo.LocalPath = setupOutput.LocalPath
	output.RepositoryInfo.BranchName = setupOutput.BranchName
	output.RepositoryInfo.DefaultBranch = setupOutput.DefaultBranch
	output.RepositoryInfo.CurrentBranch = setupOutput.CurrentBranch

	// Derive a safe directory name for the prompt file within the repo.
	repoPromptDir := filepath.Base(filepath.Clean(input.OutputDir))
	if repoPromptDir == "." {
		repoPromptDir = ""
	}
	repoPromptPath := filepath.ToSlash(filepath.Join(repoPromptDir, filepath.Base(outputPath)))

	targetPromptPath := filepath.Join(setupOutput.LocalPath, filepath.FromSlash(repoPromptPath))
	if err := os.MkdirAll(filepath.Dir(targetPromptPath), 0755); err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to create prompt file directory in repo: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}
	if err := os.WriteFile(targetPromptPath, parseFileContent, 0644); err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to write prompt file in repo branch: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}

	commitMessage := fmt.Sprintf("Add Bauer parse output for doc %s", input.DocID)
	if err := github.CommitFiles(setupOutput.LocalPath, commitMessage, []string{repoPromptPath}); err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to commit branch prompt file: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}
	if err := github.PushBranch(setupOutput.LocalPath, setupOutput.BranchName); err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to push branch prompt file: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}

	output.FinalizationInfo.CommitMessage = commitMessage
	output.FinalizationInfo.BranchPushed = true

	pinnedRef := setupOutput.BranchName
	if sha, shaErr := github.GetHeadCommitSHA(setupOutput.LocalPath); shaErr == nil && sha != "" {
		pinnedRef = sha
	} else if shaErr != nil {
		output.Warnings = append(output.Warnings, fmt.Sprintf("could not resolve HEAD SHA for pinned prompt link; falling back to branch ref: %v", shaErr))
	}

	branchBlobURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", setupOutput.Repo.Owner, setupOutput.Repo.Name, setupOutput.BranchName, repoPromptPath)
	pinnedBlobURL := fmt.Sprintf("https://github.com/%s/%s/blob/%s/%s", setupOutput.Repo.Owner, setupOutput.Repo.Name, pinnedRef, repoPromptPath)
	rawURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s", setupOutput.Repo.Owner, setupOutput.Repo.Name, pinnedRef, repoPromptPath)

	issueTitle := fmt.Sprintf("@copilot Apply Bauer parse result to %s", repo.Name)
	issueBody := fmt.Sprintf("@copilot\n\nAutomated parse result from Bauer\n\nGDoc ID: %s", input.DocID)
	if bauerResult != nil && bauerResult.ExtractionResult != nil {
		issueBody = prompt.BuildIssueDescription(
			bauerResult.ExtractionResult,
			bauerResult.Chunks,
			input.PageRefresh,
		)
	}
	issueBody = issueBody + "\n\n## Prompt Source\n\n" +
		"Use this branch-backed prompt file as the machine-readable input, together with this issue description.\n\n" +
		fmt.Sprintf("- Branch file: %s\n", branchBlobURL) +
		fmt.Sprintf("- Pinned file (commit SHA): %s\n", pinnedBlobURL) +
		fmt.Sprintf("- Raw JSON: %s\n", rawURL) +
		fmt.Sprintf("\n\n## Cleanup\n\nOnce the PR is merged or this issue is closed, please delete the branch `%s`.\n", setupOutput.BranchName)

	issueURL, issueWarning, err := createIssueWithFallback(repo.Owner, repo.Name, issueTitle, issueBody)
	if err != nil {
		output.Status = "failed"
		output.Errors = append(output.Errors, fmt.Sprintf("failed to create issue: %v", err))
		output.EndTime = time.Now()
		output.TotalDuration = output.EndTime.Sub(output.StartTime)
		return output, err
	}
	if issueWarning != "" {
		output.Warnings = append(output.Warnings, issueWarning)
	}

	output.FinalizationInfo.Issue.URL = issueURL
	output.FinalizationInfo.Issue.Title = issueTitle

	output.EndTime = time.Now()
	output.TotalDuration = output.EndTime.Sub(output.StartTime)
	output.Status = "success"

	logger.Info("workflow: parse + issue mode complete",
		"output_file", output.OutputFile,
		"issue_url", output.FinalizationInfo.Issue.URL,
		"branch", output.RepositoryInfo.BranchName,
	)

	return output, nil
}

func createIssueWithFallback(owner, repo, title, body string) (issueURL, warning string, err error) {
	issueURL, err = github.CreateIssue(owner, repo, github.CreateIssueOptions{
		Title:     title,
		Body:      body,
		Assignees: []string{"copilot"},
		Labels:    []string{"copilot", "bauer"},
	})
	if err == nil {
		return issueURL, "", nil
	}

	// Fallback 1: some repos/orgs cannot resolve/assign the "copilot" login.
	// Assign to copilot label
	issueURL, err = github.CreateIssue(owner, repo, github.CreateIssueOptions{
		Title:  title,
		Body:   body,
		Labels: []string{"copilot"},
	})
	if err == nil {
		return issueURL, "could not assign issue to 'copilot' and label set {copilot,bauer}; created issue with 'copilot' label only", nil
	}

	// Fallback 3: labels may be unavailable in the target repo.
	issueURL, err = github.CreateIssue(owner, repo, github.CreateIssueOptions{
		Title: title,
		Body:  body,
	})
	if err == nil {
		return issueURL, "could not assign issue to 'copilot' and one or more labels were unavailable; created issue without assignee/labels and kept @copilot mention in body", nil
	}

	return "", "", err
}
