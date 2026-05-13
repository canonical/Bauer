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

// CreateBranchFromDefault stashes any current changes, checks out the repo's
// default branch, pulls latest, creates a new branch, and pops the stash.
// Note: this helper intentionally does NOT commit or push — those are separate
// operations (see CommitChanges and PushBranch) so that callers can inspect
// the working tree after the branch is created but before committing.
func CreateBranchFromDefault(ctx context.Context, dir, branchName string) error {
	// Capture stash state before and after so we only pop when we actually
	// created a stash entry. "git stash push" exits 0 even when there are
	// no local changes ("No local changes to save"), so stashErr == nil is
	// not sufficient to know a stash entry was created.
	before, err := countStashEntries(ctx, dir)
	if err != nil {
		return fmt.Errorf("count stash entries before: %w", err)
	}

	stashCmd := exec.CommandContext(ctx, "git", "-C", dir, "stash", "push", "--include-untracked", "-m", "bauer-auto-stash")
	if out, err := stashCmd.CombinedOutput(); err != nil {
		// If stash itself fails, we can't safely proceed because checkout
		// may overwrite untracked files. Bail out early.
		return fmt.Errorf("git stash failed: %w, output: %s", err, out)
	}

	after, err := countStashEntries(ctx, dir)
	if err != nil {
		return fmt.Errorf("count stash entries after: %w", err)
	}
	stashed := after > before

	// Checkout default branch
	defaultBranch := getDefaultBranch(dir)
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "checkout", defaultBranch)
	if output, err := cmd.CombinedOutput(); err != nil {
		if stashed {
			_ = popStash(ctx, dir)
		}
		return fmt.Errorf("could not checkout %s: %w, output: %s", defaultBranch, err, output)
	}

	// Pull latest (non-fatal)
	_, _ = exec.CommandContext(ctx, "git", "-C", dir, "pull", "origin", defaultBranch).CombinedOutput()

	// Create new branch
	cmd = exec.CommandContext(ctx, "git", "-C", dir, "checkout", "-b", branchName)
	if output, err := cmd.CombinedOutput(); err != nil {
		if stashed {
			_ = popStash(ctx, dir)
		}
		return fmt.Errorf("could not create branch %s: %w, output: %s", branchName, err, output)
	}

	// Pop stash to restore changes onto the new branch
	if stashed {
		if err := popStash(ctx, dir); err != nil {
			return fmt.Errorf("could not restore stashed changes: %w", err)
		}
	}

	return nil
}

func countStashEntries(ctx context.Context, dir string) (int, error) {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "stash", "list")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	entries := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(entries) == 1 && entries[0] == "" {
		return 0, nil
	}
	return len(entries), nil
}

func popStash(ctx context.Context, dir string) error {
	cmd := exec.CommandContext(ctx, "git", "-C", dir, "stash", "pop")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w, output: %s", err, out)
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
