package github

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// RunGit runs a git command in the given directory and returns the trimmed output.
// Both stdout and stderr are captured and returned on error.
func RunGit(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(out))
	if err != nil {
		return trimmed, fmt.Errorf("git %s: %w (output: %s)", strings.Join(args, " "), err, trimmed)
	}
	return trimmed, nil
}
