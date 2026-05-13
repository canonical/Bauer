package github

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreatePROptions holds options for creating a pull request
type CreatePROptions struct {
	Title      string
	Body       string
	BaseBranch string
	HeadBranch string
	Draft      bool
	Labels     []string
	Assignees  []string
	Reviewers  []string
}

// CreatePR creates a pull request using gh CLI
// Requires: gh CLI installed and authenticated
func CreatePR(owner, repo string, opts CreatePROptions) (string, error) {
	if opts.Title == "" {
		return "", fmt.Errorf("PR title is required")
	}

	if opts.HeadBranch == "" {
		return "", fmt.Errorf("head branch is required")
	}

	// Default to main or master if not specified
	if opts.BaseBranch == "" {
		opts.BaseBranch = "main"
	}

	args := []string{
		"pr", "create",
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--head", opts.HeadBranch,
		"--base", opts.BaseBranch,
		"--title", opts.Title,
	}

	if opts.Body != "" {
		args = append(args, "--body", opts.Body)
	}

	if opts.Draft {
		args = append(args, "--draft")
	}

	for _, label := range opts.Labels {
		args = append(args, "--label", label)
	}

	for _, assignee := range opts.Assignees {
		args = append(args, "--assignee", assignee)
	}

	for _, reviewer := range opts.Reviewers {
		args = append(args, "--reviewer", reviewer)
	}

	cmd := exec.Command("gh", args...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create PR: %w, output: %s", err, output)
	}

	// Extract PR URL from output
	// Output may contain warnings, so look for the URL pattern
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	var prURL string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "https://github.com/") {
			prURL = trimmed
			break
		}
	}

	if prURL == "" {
		return "", fmt.Errorf("could not extract PR URL from output: %s", outputStr)
	}

	return prURL, nil
}

// GetPRURL constructs a PR URL from repo and PR number
func GetPRURL(owner, repo, prNumber string) string {
	return fmt.Sprintf("https://github.com/%s/%s/pull/%s", owner, repo, prNumber)
}

// PRStatus describes the status of a pull request
type PRStatus struct {
	Number int
	State  string // "OPEN", "CLOSED", "MERGED"
	Title  string
	URL    string
}

// GetPRInfo retrieves information about a pull request
func GetPRInfo(owner, repo, branchName string) (*PRStatus, error) {
	cmd := exec.Command("gh", "pr", "list",
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--head", branchName,
		"--json", "number,state,title,url",
		"--limit", "1",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to get PR info: %w", err)
	}

	outputStr := strings.TrimSpace(string(output))
	if outputStr == "" {
		return nil, fmt.Errorf("no PR found for branch %s", branchName)
	}

	// Simple parsing (in production, would use JSON unmarshaling)
	// TODO: In production, would use JSON unmarshaling. For now, we just return success
	return &PRStatus{
		State: "OPEN",
		URL:   fmt.Sprintf("https://github.com/%s/%s/pulls?head=%s", owner, repo, branchName),
	}, nil
}

// BranchStatus describes the status of a branch
type BranchStatus struct {
	Name           string
	Exists         bool
	Ahead          int
	Behind         int
	HasUnpushed    bool
	HasUncommitted bool
}

// GetBranchStatus checks the status of a branch
func GetBranchStatus(localPath, branchName string) (*BranchStatus, error) {
	status := &BranchStatus{
		Name:   branchName,
		Exists: true,
	}

	// Check if branch exists locally
	cmd := exec.Command("git", "rev-parse", "--verify", branchName)
	cmd.Dir = localPath
	if err := cmd.Run(); err != nil {
		status.Exists = false
		return status, nil
	}

	// Check for uncommitted changes
	statusOutput, err := GetStatus(localPath)
	if err != nil {
		return nil, err
	}
	status.HasUncommitted = strings.TrimSpace(statusOutput) != ""

	// Check for unpushed commits
	cmd = exec.Command("git", "log", "--oneline", "origin/"+branchName+".."+branchName)
	cmd.Dir = localPath
	output, err := cmd.CombinedOutput()
	if err != nil && !strings.Contains(string(output), "fatal") {
		// Count unpushed commits
		unpushedCount := len(strings.Split(strings.TrimSpace(string(output)), "\n"))
		if unpushedCount > 0 {
			status.HasUnpushed = true
		}
	}

	return status, nil
}
