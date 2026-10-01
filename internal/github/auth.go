package github

import (
	"bauer/internal/env"
	"fmt"
	"os/exec"
	"strings"
)

// GetGitHubToken retrieves a GitHub token from environment variables or gh CLI
func GetGitHubToken() (string, error) {
	if token := env.GetGoEnv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}
	if token := env.GetGoEnv("GH_TOKEN"); token != "" {
		return token, nil
	}

	// Get token from gh CLI config
	cmd := exec.Command("gh", "auth", "token")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub token from gh CLI: %w", err)
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
	if err := env.SetGoEnv("GITHUB_TOKEN", token); err != nil {
		return fmt.Errorf("failed to set GITHUB_TOKEN: %w", err)
	}

	// Also set for gh CLI
	if err := env.SetGoEnv("GH_TOKEN", token); err != nil {
		return fmt.Errorf("failed to set GH_TOKEN: %w", err)
	}

	loginCmd := exec.Command("gh", "auth", "login", "--hostname", "github.com", "--with-token")
	loginCmd.Stdin = strings.NewReader(token)
	if output, err := loginCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to log in to gh CLI with token: %w, output: %s", err, output)
	}

	// Wire git's credential helper to gh so plain "git clone"/"git push"
	// subprocesses can authenticate.
	cmd := exec.Command("gh", "auth", "setup-git")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to configure git credential helper via gh: %w, output: %s", err, output)
	}

	return nil
}

// IsGhCLIInstalled checks if gh CLI is installed
func IsGhCLIInstalled() bool {
	_, err := exec.LookPath("gh")
	return err == nil
}
