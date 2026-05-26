package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// CreateIssue creates a GitHub issue via the REST API and returns the HTML URL and issue number.
// repo must be in "owner/name" format.
// token must be a valid GitHub personal access token or fine-grained token.
func CreateIssue(ctx context.Context, token, repo, title, body string) (string, int, error) {
	if token == "" {
		return "", 0, fmt.Errorf("GitHub token is required")
	}
	if repo == "" {
		return "", 0, fmt.Errorf("repo is required (owner/name format)")
	}
	if title == "" {
		return "", 0, fmt.Errorf("issue title is required")
	}

	type issueRequest struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}

	payload, err := json.Marshal(issueRequest{Title: title, Body: body})
	if err != nil {
		return "", 0, fmt.Errorf("failed to marshal issue request: %w", err)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/issues", repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send request to GitHub API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", 0, fmt.Errorf("failed to read GitHub API response: %w", err)
	}

	if resp.StatusCode != http.StatusCreated {
		return "", 0, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, respBody)
	}

	var result struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", 0, fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	return result.HTMLURL, result.Number, nil
}
