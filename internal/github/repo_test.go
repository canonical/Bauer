package github

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// requireGit skips the test if the git binary is not available.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available; skipping")
	}
}

// runGit runs a git command in dir and fails the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v, output: %s", args, err, out)
	}
}

// createSourceRepo creates a local git repository with a single commit and
// returns its path. It is used as a clone source (git can clone from a local
// filesystem path just like a URL), so tests need no network access.
func createSourceRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init", "-b", "main")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test User")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello\n"), 0644); err != nil {
		t.Fatalf("failed to write file in source repo: %v", err)
	}
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial commit")
	return dir
}

// createBrokenRepo creates a directory containing a .git folder that is missing
// the essential metadata (HEAD, config, index), mimicking a partial/interrupted
// clone that git refuses to recognize as a repository.
func createBrokenRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	for _, sub := range []string{"objects", "refs", "hooks", "info", "logs"} {
		if err := os.MkdirAll(filepath.Join(dir, ".git", sub), 0755); err != nil {
			t.Fatalf("failed to create broken .git subdir %q: %v", sub, err)
		}
	}
	return dir
}

func TestIsGitRepo(t *testing.T) {
	requireGit(t)

	t.Run("valid repo returns true", func(t *testing.T) {
		src := createSourceRepo(t)
		if !isGitRepo(src) {
			t.Fatalf("expected isGitRepo to return true for a valid repo")
		}
	})

	t.Run("broken .git returns false", func(t *testing.T) {
		broken := createBrokenRepo(t)
		if isGitRepo(broken) {
			t.Fatalf("expected isGitRepo to return false for a broken .git directory")
		}
	})

	t.Run("no .git returns false", func(t *testing.T) {
		if isGitRepo(t.TempDir()) {
			t.Fatalf("expected isGitRepo to return false for a directory without .git")
		}
	})

	t.Run("nonexistent path returns false", func(t *testing.T) {
		if isGitRepo(filepath.Join(t.TempDir(), "does-not-exist")) {
			t.Fatalf("expected isGitRepo to return false for a nonexistent path")
		}
	})
}

func TestCloneOrUpdateRepo_FreshClone(t *testing.T) {
	requireGit(t)

	src := createSourceRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")

	repo := &Repository{Owner: "test", Name: "repo", HTTPURL: src}
	if err := CloneOrUpdateRepo(repo, dest); err != nil {
		t.Fatalf("CloneOrUpdateRepo returned error on fresh clone: %v", err)
	}

	if !isGitRepo(dest) {
		t.Fatalf("expected destination to be a valid git repo after clone")
	}
	if repo.LocalPath != dest {
		t.Fatalf("expected repo.LocalPath = %q, got %q", dest, repo.LocalPath)
	}
	if _, err := os.Stat(filepath.Join(dest, "README.md")); err != nil {
		t.Fatalf("expected cloned file README.md to exist: %v", err)
	}
}

func TestCloneOrUpdateRepo_BrokenRepoReclone(t *testing.T) {
	requireGit(t)

	src := createSourceRepo(t)
	broken := createBrokenRepo(t)

	// Sanity check: the broken directory is not recognized as a repo.
	if isGitRepo(broken) {
		t.Fatalf("precondition failed: broken directory should not be a git repo")
	}

	repo := &Repository{Owner: "test", Name: "repo", HTTPURL: src}
	if err := CloneOrUpdateRepo(repo, broken); err != nil {
		t.Fatalf("CloneOrUpdateRepo returned error re-cloning broken repo: %v", err)
	}

	if !isGitRepo(broken) {
		t.Fatalf("expected broken directory to be re-cloned into a valid repo")
	}
	if _, err := os.Stat(filepath.Join(broken, "README.md")); err != nil {
		t.Fatalf("expected re-cloned file README.md to exist: %v", err)
	}
}

func TestCloneOrUpdateRepo_UpdateExisting(t *testing.T) {
	requireGit(t)

	src := createSourceRepo(t)
	dest := filepath.Join(t.TempDir(), "clone")

	// First clone.
	repo := &Repository{Owner: "test", Name: "repo", HTTPURL: src}
	if err := CloneOrUpdateRepo(repo, dest); err != nil {
		t.Fatalf("initial clone failed: %v", err)
	}

	// Second call on the existing valid repo should update without error.
	if err := CloneOrUpdateRepo(repo, dest); err != nil {
		t.Fatalf("CloneOrUpdateRepo returned error updating existing repo: %v", err)
	}

	if !isGitRepo(dest) {
		t.Fatalf("expected destination to remain a valid git repo after update")
	}
}
