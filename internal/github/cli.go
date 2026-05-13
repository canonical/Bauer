package github

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ReadRemoteFromGitConfig reads the origin URL from the git config in dir
// and returns the parsed owner and repo name.
func ReadRemoteFromGitConfig(ctx context.Context, dir string) (owner, repo string, err error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "remote", "get-url", "origin")
	out, err := cmd.Output()
	if err != nil {
		return "", "", fmt.Errorf("could not read git remote 'origin': %w (is this a git repo?)", err)
	}
	return parseRepoFromURL(strings.TrimSpace(string(out)))
}

func parseRepoFromURL(rawURL string) (owner, repo string, err error) {
	// Handle HTTPS: https://github.com/owner/repo.git
	if strings.HasPrefix(rawURL, "https://github.com/") {
		parts := strings.TrimPrefix(rawURL, "https://github.com/")
		parts = strings.TrimSuffix(parts, ".git")
		segments := strings.Split(parts, "/")
		if len(segments) >= 2 {
			return segments[0], segments[1], nil
		}
	}
	// Handle SSH: git@github.com:owner/repo.git
	if strings.HasPrefix(rawURL, "git@github.com:") {
		parts := strings.TrimPrefix(rawURL, "git@github.com:")
		parts = strings.TrimSuffix(parts, ".git")
		segments := strings.Split(parts, "/")
		if len(segments) >= 2 {
			return segments[0], segments[1], nil
		}
	}
	return "", "", fmt.Errorf("could not parse owner/repo from remote URL: %s", rawURL)
}

// CreateAndPushBranch creates a new branch from the default branch, commits
// any working-directory changes, and pushes the branch to origin.
func CreateAndPushBranch(ctx context.Context, dir, branchName string) error {
	// Stash any current changes so we can checkout the default branch cleanly.
	stashCmd := exec.CommandContext(ctx, "git", "-C", dir, "stash", "push", "-m", "bauer-auto-stash")
	_, stashErr := stashCmd.Output()
	stashed := stashErr == nil

	// Checkout default branch
	defaultBranch := getDefaultBranch(dir)
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "checkout", defaultBranch)
	if output, err := cmd.CombinedOutput(); err != nil {
		if stashed {
			exec.CommandContext(ctx, "git", "-C", dir, "stash", "pop").Run()
		}
		return fmt.Errorf("could not checkout %s: %w, output: %s", defaultBranch, err, output)
	}

	// Pull latest (non-fatal)
	exec.CommandContext(ctx, "git", "-C", dir, "pull", "origin", defaultBranch).CombinedOutput()

	// Create new branch
	cmd = exec.CommandContext(ctx, "git", "-C", dir, "checkout", "-b", branchName)
	if output, err := cmd.CombinedOutput(); err != nil {
		if stashed {
			exec.CommandContext(ctx, "git", "-C", dir, "stash", "pop").Run()
		}
		return fmt.Errorf("could not create branch %s: %w, output: %s", branchName, err, output)
	}

	// Pop stash to restore changes onto the new branch
	if stashed {
		cmd = exec.CommandContext(ctx, "git", "-C", dir, "stash", "pop")
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("could not restore stashed changes: %w, output: %s", err, output)
		}
	}

	return nil
}

// CreateIssue creates a GitHub issue and returns its URL and number.
func CreateIssue(ctx context.Context, owner, repo, title, body string) (string, int, error) {
	args := []string{
		"issue", "create",
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--title", title,
		"--body", body,
	}

	cmd := exec.CommandContext(ctx, "gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", 0, fmt.Errorf("failed to create issue: %w, output: %s", err, output)
	}

	// Extract issue URL from output
	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	var issueURL string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "https://github.com/") && strings.Contains(trimmed, "/issues/") {
			issueURL = trimmed
			break
		}
	}

	if issueURL == "" {
		return "", 0, fmt.Errorf("could not extract issue URL from output: %s", outputStr)
	}

	// Extract issue number from URL
	parts := strings.Split(issueURL, "/")
	if len(parts) > 0 {
		numStr := parts[len(parts)-1]
		if num, err := strconv.Atoi(numStr); err == nil {
			return issueURL, num, nil
		}
	}

	return issueURL, 0, nil
}
