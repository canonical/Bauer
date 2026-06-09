package github

import (
	"fmt"
	"os/exec"
	"strings"
)

// CreateIssueOptions holds options for creating an issue.
type CreateIssueOptions struct {
	Title     string
	Body      string
	Labels    []string
	Assignees []string
}

// CreateIssue creates a GitHub issue using gh CLI.
func CreateIssue(owner, repo string, opts CreateIssueOptions) (string, error) {
	if opts.Title == "" {
		return "", fmt.Errorf("issue title is required")
	}

	args := []string{
		"issue", "create",
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--title", opts.Title,
	}

	if opts.Body != "" {
		args = append(args, "--body", opts.Body)
	}

	for _, label := range opts.Labels {
		args = append(args, "--label", label)
	}

	for _, assignee := range opts.Assignees {
		args = append(args, "--assignee", assignee)
	}

	cmd := exec.Command("gh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to create issue: %w, output: %s", err, output)
	}

	outputStr := string(output)
	lines := strings.Split(outputStr, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "https://github.com/") {
			return trimmed, nil
		}
	}

	return "", fmt.Errorf("could not extract issue URL from output: %s", outputStr)
}

// AddIssueComment posts a comment to an existing issue.
func AddIssueComment(owner, repo, issueNumber, body string) error {
	if owner == "" || repo == "" || issueNumber == "" {
		return fmt.Errorf("owner, repo, and issueNumber are required")
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("comment body cannot be empty")
	}

	cmd := exec.Command(
		"gh", "issue", "comment", issueNumber,
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--body", body,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add issue comment: %w, output: %s", err, output)
	}

	return nil
}

// ExtractIssueNumberFromURL parses issue URL like
// https://github.com/owner/repo/issues/123 and returns "123".
func ExtractIssueNumberFromURL(issueURL string) (string, error) {
	issueURL = strings.TrimSpace(issueURL)
	if issueURL == "" {
		return "", fmt.Errorf("issue URL is empty")
	}
	parts := strings.Split(issueURL, "/")
	if len(parts) == 0 {
		return "", fmt.Errorf("invalid issue URL: %s", issueURL)
	}
	n := parts[len(parts)-1]
	if n == "" {
		return "", fmt.Errorf("invalid issue URL: %s", issueURL)
	}
	return n, nil
}
