package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// CreateIssue creates a GitHub issue via the REST API and returns the HTML URL.
// repo must be in "owner/name" format.
// token must be a valid GitHub personal access token or fine-grained token.
func CreateIssue(ctx context.Context, token, repo, title, body string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("GitHub token is required")
	}
	if repo == "" {
		return "", fmt.Errorf("repo is required (owner/name format)")
	}
	parts := strings.SplitN(repo, "/", 3)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", fmt.Errorf("repo must be in \"owner/name\" format, got %q", repo)
	}
	if title == "" {
		return "", fmt.Errorf("issue title is required")
	}

	type issueRequest struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	payload, err := json.Marshal(issueRequest{Title: title, Body: body})
	if err != nil {
		return "", fmt.Errorf("failed to marshal issue request: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/issues", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read GitHub API response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	return result.HTMLURL, nil
}
