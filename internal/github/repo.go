package github

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Repository struct {
	Owner     string
	Name      string
	LocalPath string
	HTTPURL   string
}

// ParseGitHubRepo parses a GitHub repo string in various formats
// Supports: "owner/repo", "https://github.com/owner/repo", "git@github.com:owner/repo.git"
func ParseGitHubRepo(input string) (*Repository, error) {
	var owner, name string

	// Handle HTTPS URL
	if strings.HasPrefix(input, "https://github.com/") {
		parts := strings.TrimPrefix(input, "https://github.com/")
		parts = strings.TrimSuffix(parts, ".git")
		segments := strings.Split(parts, "/")
		if len(segments) < 2 {
			return nil, fmt.Errorf("invalid GitHub URL: %s", input)
		}
		owner, name = segments[0], segments[1]
	} else if strings.HasPrefix(input, "git@github.com:") {
		// Handle SSH URL
		parts := strings.TrimPrefix(input, "git@github.com:")
		parts = strings.TrimSuffix(parts, ".git")
		segments := strings.Split(parts, "/")
		if len(segments) < 2 {
			return nil, fmt.Errorf("invalid GitHub SSH URL: %s", input)
		}
		owner, name = segments[0], segments[1]
	} else if strings.Contains(input, "/") && !strings.Contains(input, "://") {
		// Handle "owner/repo" format
		segments := strings.Split(input, "/")
		if len(segments) != 2 {
			return nil, fmt.Errorf("invalid repo format: %s, expected 'owner/repo'", input)
		}
		owner, name = segments[0], segments[1]
	} else {
		return nil, fmt.Errorf("invalid GitHub repo format: %s", input)
	}

	return &Repository{
		Owner:   owner,
		Name:    name,
		HTTPURL: fmt.Sprintf("https://github.com/%s/%s.git", owner, name),
	}, nil
}

// CloneOrUpdateRepo clones or updates a repository at the specified local path
func CloneOrUpdateRepo(repo *Repository, localPath string) error {
	info, err := os.Stat(localPath)

	// If path doesn't exist, clone
	if os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(localPath), 0755); err != nil {
			return fmt.Errorf("failed to create parent directory: %w", err)
		}

		cmd := exec.Command("git", "clone", repo.HTTPURL, localPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to clone repo: %w, output: %s", err, output)
		}
		repo.LocalPath = localPath
		return nil
	}

	if err != nil {
		return fmt.Errorf("error checking path: %w", err)
	}

	// If path exists but is not a directory, error
	if !info.IsDir() {
		return fmt.Errorf("path exists but is not a directory: %s", localPath)
	}

	// If directory exists and is a git repo, pull latest
	if isGitRepo(localPath) {
		// Ensure origin points to the expected URL before fetching.
		remoteCmd := exec.Command("git", "remote", "get-url", "origin")
		remoteCmd.Dir = localPath
		remoteOut, remoteErr := remoteCmd.CombinedOutput()
		if remoteErr != nil {
			// origin does not exist — add it.
			addCmd := exec.Command("git", "remote", "add", "origin", repo.HTTPURL)
			addCmd.Dir = localPath
			if out, err := addCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to add origin remote: %w, output: %s", err, out)
			}
		} else if strings.TrimSpace(string(remoteOut)) != repo.HTTPURL {
			// origin exists but points elsewhere — update it.
			setCmd := exec.Command("git", "remote", "set-url", "origin", repo.HTTPURL)
			setCmd.Dir = localPath
			if out, err := setCmd.CombinedOutput(); err != nil {
				return fmt.Errorf("failed to update origin remote: %w, output: %s", err, out)
			}
		}

		cmd := exec.Command("git", "fetch", "origin")
		cmd.Dir = localPath
		if output, err := cmd.CombinedOutput(); err != nil {
			// The local repo may be corrupted (e.g. "pack has unresolved deltas").
			// Remove it and fall back to a fresh clone.
			if removeErr := os.RemoveAll(localPath); removeErr != nil {
				return fmt.Errorf("failed to fetch from remote: %w, output: %s (also failed to remove corrupted repo: %v)", err, output, removeErr)
			}
			cloneCmd := exec.Command("git", "clone", repo.HTTPURL, localPath)
			if cloneOut, cloneErr := cloneCmd.CombinedOutput(); cloneErr != nil {
				return fmt.Errorf("failed to fetch from remote: %w, output: %s (re-clone also failed: %v, output: %s)", err, output, cloneErr, cloneOut)
			}
			repo.LocalPath = localPath
			return nil
		}

		cmd = exec.Command("git", "pull", "origin", getDefaultBranch(localPath))
		cmd.Dir = localPath
		if _, err := cmd.CombinedOutput(); err != nil {
			// Non-fatal: might be on a different branch
			fmt.Printf("Warning: failed to pull latest: %v\n", err)
		}
		repo.LocalPath = localPath
		return nil
	}

	return fmt.Errorf("path exists but is not a git repository: %s", localPath)
}

// GetDefaultBranch returns the default branch name (main or master)
func GetDefaultBranch(localPath string) (string, error) {
	name := getDefaultBranch(localPath)
	return name, nil
}

// CreateFeatureBranch creates a new feature branch and checks it out
func CreateFeatureBranch(localPath, branchName string) error {
	// Checkout to default branch
	defaultBranch := getDefaultBranch(localPath)
	cmd := exec.Command("git", "checkout", defaultBranch)
	cmd.Dir = localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout to %s: %w, output: %s", defaultBranch, err, output)
	}

	// Pull latest changes
	cmd = exec.Command("git", "pull", "origin", defaultBranch)
	cmd.Dir = localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to pull latest from %s: %w, output: %s", defaultBranch, err, output)
	}

	// Create new branch
	cmd = exec.Command("git", "checkout", "-b", branchName)
	cmd.Dir = localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create branch %s: %w, output: %s", branchName, err, output)
	}

	return nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(localPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = localPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// GetStatus returns git status in machine-readable format
func GetStatus(localPath string) (string, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = localPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get git status: %w", err)
	}
	return string(output), nil
}

// CommitChanges stages all changes and commits with a message
func CommitChanges(localPath, message string) error {
	// Stage all changes
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage changes: %w, output: %s", err, output)
	}

	// Exclude specific files from commit
	excludeFiles := []string{
		"bauer-doc-suggestions.json",
		"bauer-output/",
	}
	for _, file := range excludeFiles {
		cmd := exec.Command("git", "reset", "HEAD", file)
		cmd.Dir = localPath
		// Ignore error if file doesn't exist
		cmd.CombinedOutput()
	}

	// Check if there are changes to commit
	status, err := GetStatus(localPath)
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) == "" {
		return fmt.Errorf("no changes to commit")
	}

	// Commit
	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit changes: %w, output: %s", err, output)
	}

	return nil
}

// CommitFiles stages only the provided files and commits them with a message.
func CommitFiles(localPath, message string, files []string) error {
	if len(files) == 0 {
		return fmt.Errorf("no files provided to commit")
	}

	args := []string{"add", "--"}
	args = append(args, files...)
	cmd := exec.Command("git", args...)
	cmd.Dir = localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage files: %w, output: %s", err, output)
	}

	cmd = exec.Command("git", "diff", "--cached", "--name-only")
	cmd.Dir = localPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to inspect staged files: %w, output: %s", err, output)
	}
	if strings.TrimSpace(string(output)) == "" {
		return fmt.Errorf("no staged changes to commit")
	}

	cmd = exec.Command("git", "commit", "-m", message)
	cmd.Dir = localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to commit files: %w, output: %s", err, output)
	}

	return nil
}

// GetHeadCommitSHA returns the SHA of HEAD in the given repository.
func GetHeadCommitSHA(localPath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = localPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to read HEAD commit SHA: %w, output: %s", err, output)
	}
	return strings.TrimSpace(string(output)), nil
}

// PushBranch pushes the specified branch to remote
func PushBranch(localPath, branchName string) error {
	cmd := exec.Command("git", "push", "origin", branchName)
	cmd.Dir = localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to push branch %s: %w, output: %s", branchName, err, output)
	}
	return nil
}

// DeleteLocalBranch deletes a local branch (without force)
func DeleteLocalBranch(localPath, branchName string) error {
	cmd := exec.Command("git", "branch", "-d", branchName)
	cmd.Dir = localPath
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to delete local branch %s: %w, output: %s", branchName, err, output)
	}
	return nil
}

// Helper functions

func isGitRepo(path string) bool {
	gitDir := filepath.Join(path, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

func getDefaultBranch(localPath string) string {
	// Get branch from origin/HEAD
	cmd := exec.Command("git", "symbolic-ref", "refs/remotes/origin/HEAD")
	cmd.Dir = localPath
	output, err := cmd.CombinedOutput()
	if err == nil {
		ref := strings.TrimSpace(string(output))
		// Format: "refs/remotes/origin/main" or "refs/remotes/origin/master"
		parts := strings.Split(ref, "/")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}

	cmd = exec.Command("git", "rev-parse", "--verify", "origin/main")
	cmd.Dir = localPath
	if err := cmd.Run(); err == nil {
		return "main"
	}

	return "master"
}
