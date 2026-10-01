package github

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeGhScript is a stand-in "gh" binary that records invocations instead of
// touching the real GitHub API. It distinguishes "auth login" from
// "auth setup-git" via $2, and its failure behaviour is toggled with env vars
// so tests can exercise both success and error paths.
const fakeGhScript = `#!/bin/sh
case "$2" in
  login)
    echo "login-called" >> "$GH_FAKE_LOG"
    if [ "$GH_FAKE_LOGIN_FAIL" = "1" ]; then
      echo "simulated login failure" >&2
      exit 1
    fi
    cat > "$GH_FAKE_TOKEN_FILE"
    exit 0
    ;;
  setup-git)
    echo "setup-git-called" >> "$GH_FAKE_LOG"
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
	log := string(logBytes)
	loginIdx := strings.Index(log, "login-called")
	setupIdx := strings.Index(log, "setup-git-called")
	if loginIdx == -1 || setupIdx == -1 {
		t.Fatalf("expected both gh auth login and gh auth setup-git to be called, got log: %q", log)
	}
	if loginIdx > setupIdx {
		t.Fatalf("expected gh auth login to run before gh auth setup-git, got log: %q", log)
	}

	gotToken, err := os.ReadFile(tokenFile)
	if err != nil {
		t.Fatalf("failed to read captured token: %v", err)
	}
	if strings.TrimSpace(string(gotToken)) != token {
		// Proves the token is delivered via stdin, not a visible argv entry.
		t.Fatalf("expected token %q piped to stdin of gh auth login, got %q", token, gotToken)
	}
}

func TestSetupGitHubAuth_LoginFailure(t *testing.T) {
	installFakeGh(t)
	t.Setenv("GH_FAKE_LOGIN_FAIL", "1")

	err := SetupGitHubAuth("test-token-123")
	if err == nil {
		t.Fatal("expected error when gh auth login fails")
	}
	if !strings.Contains(err.Error(), "failed to log in to gh CLI with token") {
		t.Fatalf("expected login-failure error, got: %v", err)
	}
}

func TestSetupGitHubAuth_SetupGitFailure(t *testing.T) {
	logFile, _ := installFakeGh(t)
	t.Setenv("GH_FAKE_SETUP_FAIL", "1")

	err := SetupGitHubAuth("test-token-123")
	if err == nil {
		t.Fatal("expected error when gh auth setup-git fails")
	}
	if !strings.Contains(err.Error(), "failed to configure git credential helper via gh") {
		t.Fatalf("expected setup-git-failure error, got: %v", err)
	}

	logBytes, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("failed to read fake gh log: %v", err)
	}
	if !strings.Contains(string(logBytes), "login-called") {
		t.Fatalf("expected gh auth login to have run before the setup-git failure, got log: %q", logBytes)
	}
}

func TestSetupGitHubAuth_EmptyToken(t *testing.T) {
	if err := SetupGitHubAuth(""); err == nil {
		t.Fatal("expected error for empty token")
	}
}
