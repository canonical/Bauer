package github

import (
	"encoding/base64"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CreateParseResultPROptions configures remote PR creation from parse output.
type CreateParseResultPROptions struct {
	Owner        string
	Repo         string
	BranchPrefix string
	BaseBranch   string
	PRTitle      string
	PRBody       string
	FilePath     string
	FileContent  []byte
	CommitMsg    string
}

// CreateParseResultPR creates a branch, commits parse-result content, and opens a PR
// directly against the target repository using gh API (no local clone required).
func CreateParseResultPR(opts CreateParseResultPROptions) (prURL string, branchName string, err error) {
	if opts.Owner == "" || opts.Repo == "" {
		return "", "", fmt.Errorf("owner and repo are required")
	}
	if len(opts.FileContent) == 0 {
		return "", "", fmt.Errorf("file content cannot be empty")
	}
	if opts.FilePath == "" {
		opts.FilePath = "bauer-output/bauer-parse-result.json"
	}
	if opts.CommitMsg == "" {
		opts.CommitMsg = "Add bauer parse result"
	}

	if opts.BaseBranch == "" {
		defaultBranch, getErr := GetDefaultBranchRemote(opts.Owner, opts.Repo)
		if getErr != nil {
			return "", "", fmt.Errorf("failed to resolve base branch: %w", getErr)
		}
		opts.BaseBranch = defaultBranch
	}

	if opts.BranchPrefix == "" {
		opts.BranchPrefix = "bauer"
	}

	branchName = fmt.Sprintf("%s/parse-result-%d", opts.BranchPrefix, time.Now().Unix())

	baseSHA, err := getBranchHeadSHA(opts.Owner, opts.Repo, opts.BaseBranch)
	if err != nil {
		return "", "", fmt.Errorf("failed to read base branch sha: %w", err)
	}

	if err := createBranchRef(opts.Owner, opts.Repo, branchName, baseSHA); err != nil {
		return "", "", fmt.Errorf("failed to create remote branch: %w", err)
	}

	encodedContent := base64.StdEncoding.EncodeToString(opts.FileContent)
	if err := putFileOnBranch(opts.Owner, opts.Repo, branchName, opts.FilePath, opts.CommitMsg, encodedContent); err != nil {
		return "", "", fmt.Errorf("failed to commit parse result file: %w", err)
	}

	prOpts := CreatePROptions{
		Title:      opts.PRTitle,
		Body:       opts.PRBody,
		HeadBranch: branchName,
		BaseBranch: opts.BaseBranch,
	}

	prURL, err = CreatePR(opts.Owner, opts.Repo, prOpts)
	if err != nil {
		return "", branchName, fmt.Errorf("failed to create PR: %w", err)
	}

	return prURL, branchName, nil
}

// GetDefaultBranchRemote resolves the default branch for a repository via gh api.
func GetDefaultBranchRemote(owner, repo string) (string, error) {
	cmd := exec.Command("gh", "api", fmt.Sprintf("repos/%s/%s", owner, repo), "--jq", ".default_branch")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh api failed: %w, output: %s", err, output)
	}
	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("empty default branch response")
	}
	return branch, nil
}

func getBranchHeadSHA(owner, repo, branch string) (string, error) {
	cmd := exec.Command(
		"gh", "api",
		fmt.Sprintf("repos/%s/%s/git/ref/heads/%s", owner, repo, branch),
		"--jq", ".object.sha",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh api failed: %w, output: %s", err, output)
	}
	sha := strings.TrimSpace(string(output))
	if sha == "" {
		return "", fmt.Errorf("empty sha response for branch %s", branch)
	}
	return sha, nil
}

func createBranchRef(owner, repo, branchName, sha string) error {
	cmd := exec.Command(
		"gh", "api", "-X", "POST",
		fmt.Sprintf("repos/%s/%s/git/refs", owner, repo),
		"-f", "ref=refs/heads/"+branchName,
		"-f", "sha="+sha,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh api failed: %w, output: %s", err, output)
	}
	return nil
}

func putFileOnBranch(owner, repo, branch, path, message, encodedContent string) error {
	cmd := exec.Command(
		"gh", "api", "-X", "PUT",
		fmt.Sprintf("repos/%s/%s/contents/%s", owner, repo, path),
		"-f", "message="+message,
		"-f", "content="+encodedContent,
		"-f", "branch="+branch,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh api failed: %w, output: %s", err, output)
	}
	return nil
}
