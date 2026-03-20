// Package git provides git and GitHub CLI operations with timeout support.
package git

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultCmdTimeout = 30 * time.Second

// ExecResult holds the output of a command execution.
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// RunGit executes a git command in the given directory with a timeout.
func RunGit(ctx context.Context, dir string, args ...string) (ExecResult, error) {
	return runCmd(ctx, dir, "git", args...)
}

// RunGH executes a GitHub CLI command in the given directory with a timeout.
func RunGH(ctx context.Context, dir string, args ...string) (ExecResult, error) {
	return runCmd(ctx, dir, "gh", args...)
}

func runCmd(ctx context.Context, dir, name string, args ...string) (ExecResult, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultCmdTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := ExecResult{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}

	if exitErr, ok := err.(*exec.ExitError); ok {
		result.ExitCode = exitErr.ExitCode()
		return result, fmt.Errorf("command %q failed (exit %d): %s", name+" "+strings.Join(args, " "), result.ExitCode, result.Stderr)
	}

	if err != nil {
		return result, fmt.Errorf("executing %q: %w", name, err)
	}

	return result, nil
}
