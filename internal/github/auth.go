package github

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GetGitHubToken retrieves a GitHub token from environment variables or gh CLI.
// Resolution order:
//  1. GitHub App (if GITHUB_APP_ID is set)
//  2. PAT env vars (BAUER_GITHUB_TOKEN, GITHUB_TOKEN, GH_TOKEN)
//  3. gh auth token CLI
func GetGitHubToken() (string, error) {
	// 1. Try GitHub App installation token
	if os.Getenv("GITHUB_APP_ID") != "" {
		token, err := generateAppInstallationToken()
		if err != nil {
			return "", fmt.Errorf("GitHub App auth failed: %w", err)
		}
		return token, nil
	}

	// 2. PAT env vars
	for _, env := range []string{"BAUER_GITHUB_TOKEN", "GITHUB_TOKEN", "GH_TOKEN"} {
		if v := os.Getenv(env); v != "" {
			return v, nil
		}
	}

	// 3. Get token from gh CLI config
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

// generateAppInstallationToken generates a GitHub App installation access token
// using a signed JWT exchanged for an installation token via the GitHub REST API.
func generateAppInstallationToken() (string, error) {
	appIDStr := os.Getenv("GITHUB_APP_ID")
	appID, err := strconv.ParseInt(appIDStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid GITHUB_APP_ID: %w", err)
	}

	installIDStr := os.Getenv("GITHUB_APP_INSTALLATION_ID")
	installID, err := strconv.ParseInt(installIDStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid GITHUB_APP_INSTALLATION_ID: %w", err)
	}

	// Load private key from env or file
	var pemData []byte
	if keyPath := os.Getenv("GITHUB_APP_PRIVATE_KEY_PATH"); keyPath != "" {
		pemData, err = os.ReadFile(keyPath)
		if err != nil {
			return "", fmt.Errorf("reading GITHUB_APP_PRIVATE_KEY_PATH: %w", err)
		}
	} else if keyContent := os.Getenv("GITHUB_APP_PRIVATE_KEY"); keyContent != "" {
		// Replace literal \n with newlines (common in env var storage)
		pemData = []byte(strings.ReplaceAll(keyContent, `\n`, "\n"))
	} else {
		return "", fmt.Errorf("set GITHUB_APP_PRIVATE_KEY or GITHUB_APP_PRIVATE_KEY_PATH")
	}

	// Parse RSA private key
	block, _ := pem.Decode(pemData)
	if block == nil {
		return "", fmt.Errorf("failed to decode PEM block from GitHub App private key")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return "", fmt.Errorf("parsing GitHub App private key: %w", err)
	}

	// Create JWT (signed with RS256, valid 10 min)
	now := time.Now()
	claims := jwt.RegisteredClaims{
		IssuedAt:  jwt.NewNumericDate(now.Add(-60 * time.Second)), // 60s in the past to handle clock skew
		ExpiresAt: jwt.NewNumericDate(now.Add(10 * time.Minute)),
		Issuer:    strconv.FormatInt(appID, 10),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	jwtStr, err := jwtToken.SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("signing GitHub App JWT: %w", err)
	}

	// Exchange JWT for installation access token
	url := fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installID)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", fmt.Errorf("creating installation token request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+jwtStr)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("exchanging JWT for installation token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("installation token exchange failed (status %d)", resp.StatusCode)
	}

	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("parsing installation token response: %w", err)
	}
	if result.Token == "" {
		return "", fmt.Errorf("empty installation token in response")
	}
	return result.Token, nil
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
