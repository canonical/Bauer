package github

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGhScript is a stand-in "gh" binary that records invocations and the
// GH_TOKEN it was handed instead of touching the real GitHub API. Its
// failure behaviour is toggled with an env var so tests can exercise both
// success and error paths.
const fakeGhScript = `#!/bin/sh
case "$2" in
  setup-git)
    echo "setup-git-called" >> "$GH_FAKE_LOG"
    echo "$GH_TOKEN" > "$GH_FAKE_TOKEN_FILE"
    if [ "$GH_FAKE_SETUP_FAIL" = "1" ]; then
      echo "simulated setup-git failure" >&2
      exit 1
    fi
    exit 0
    ;;
  *)
    echo "unexpected gh invocation: $*" >&2
    exit 1
    ;;
esac
`

// installFakeGh puts a stub "gh" executable at the front of PATH and points
// its log/token-capture files at paths under t.TempDir().
func installFakeGh(t *testing.T) (logFile, tokenFile string) {
	t.Helper()
	dir := t.TempDir()
	scriptPath := filepath.Join(dir, "gh")
	if err := os.WriteFile(scriptPath, []byte(fakeGhScript), 0755); err != nil {
		t.Fatalf("failed to write fake gh script: %v", err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	logFile = filepath.Join(dir, "log")
	tokenFile = filepath.Join(dir, "token")
	t.Setenv("GH_FAKE_LOG", logFile)
	t.Setenv("GH_FAKE_TOKEN_FILE", tokenFile)
	return logFile, tokenFile
}

func TestSetupGitHubAuth_Success(t *testing.T) {
	logFile, tokenFile := installFakeGh(t)

	const token = "test-token-123"
	if err := SetupGitHubAuth(token); err != nil {
		t.Fatalf("SetupGitHubAuth returned error: %v", err)
	}

	logBytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read fake gh log: %v", err)
	}
	if !strings.Contains(string(logBytes), "setup-git-called") {
		t.Fatalf("expected gh auth setup-git to be called, got log: %q", logBytes)
	}

	// Proves the token is forwarded via GH_TOKEN env for this one invocation,
	// without depending on a persisted/validated gh login (no network call).
	gotToken, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("failed to read captured token: %v", err)
	}
	if strings.TrimSpace(string(gotToken)) != token {
		t.Fatalf("expected token %q forwarded via GH_TOKEN to gh auth setup-git, got %q", token, gotToken)
	}
}

func TestSetupGitHubAuth_SetupGitFailure(t *testing.T) {
	installFakeGh(t)
	t.Setenv("GH_FAKE_SETUP_FAIL", "1")

	err := SetupGitHubAuth("test-token-123")
	if err == nil {
		t.Fatal("expected error when gh auth setup-git fails")
	}
	if !strings.Contains(err.Error(), "failed to configure git credential helper via gh") {
		t.Fatalf("expected setup-git-failure error, got: %v", err)
	}
}

func TestSetupGitHubAuth_EmptyToken(t *testing.T) {
	if err := SetupGitHubAuth(""); err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestGitCommand_ForwardsTokenWhenAvailable(t *testing.T) {
	t.Setenv("APP_GITHUB_TOKEN", "forwarded-token")

	cmd := gitCommand("", "status")
	if cmd.Env == nil {
		t.Fatal("expected gitCommand to set a custom env when a token is available")
	}
	var sawGHToken, sawGitHubToken bool
	for _, kv := range cmd.Env {
		if kv == "GH_TOKEN=forwarded-token" {
			sawGHToken = true
		}
		if kv == "GITHUB_TOKEN=forwarded-token" {
			sawGitHubToken = true
		}
	}
	if !sawGHToken || !sawGitHubToken {
		t.Fatalf("expected GH_TOKEN and GITHUB_TOKEN to be forwarded in cmd.Env, got: %v", cmd.Env)
	}
}

func TestGitCommand_NoTokenLeavesEnvUnset(t *testing.T) {
	t.Setenv("APP_GITHUB_TOKEN", "")
	t.Setenv("APP_GH_TOKEN", "")
	t.Setenv("PATH", "/nonexistent") // make "gh auth token" fail fast, no real CLI involved

	cmd := gitCommand("", "status")
	if cmd.Env != nil {
		t.Fatalf("expected gitCommand to leave cmd.Env nil (inherit default) when no token is available, got: %v", cmd.Env)
	}
}
