package github

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetGitHubToken retrieves a GitHub token from environment variables or gh CLI.
// Resolution order: BAUER_GITHUB_TOKEN → GITHUB_TOKEN → GH_TOKEN → gh auth token
func GetGitHubToken() (string, error) {
	for _, env := range []string{"BAUER_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if token := os.Getenv(env); token != "" {
			return token, nil
		}
	}

	// Fall back to gh CLI
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("no GitHub token found (tried BAUER_GITHUB_TOKEN, GITHUB_TOKEN, GH_TOKEN, and gh CLI): %w (run 'gh auth login' or set one of the env vars)", err)
	}

	token := strings.TrimSpace(string(output))
	if token == "" {
		return "", fmt.Errorf("no GitHub token found in environment or gh CLI config")
	}

	return token, nil
}

// ValidateGitHubAuth checks if GitHub authentication is configured
func ValidateGitHubAuth() error {
	// Get token
	_, err := GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub authentication not configured: %w", err)
	}

	// Authenticate token
	cmd := exec.Command("gh", "auth", "status")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to verify GitHub authentication: %w, output: %s", err, output)
	}

	return nil
}

// SetupGitHubAuth configures GitHub authentication for the current shell session
func SetupGitHubAuth(token string) error {
	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Set environment variable for this process and child processes
	if err := os.Setenv("GITHUB_TOKEN", token); err != nil {
		return fmt.Errorf("failed to set GITHUB_TOKEN: %w", err)
	}

	// Also set for gh CLI
	if err := os.Setenv("GH_TOKEN", token); err != nil {
		return fmt.Errorf("failed to set GH_TOKEN: %w", err)
	}

	return nil
}

// IsGhCLIInstalled checks if gh CLI is installed
func IsGhCLIInstalled() bool {
	cmd := exec.Command("which", "gh")
	return cmd.Run() == nil
}
