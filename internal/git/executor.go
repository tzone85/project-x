package git

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"time"
)

// CommandRunner executes shell commands. Extracted as an interface for testing.
type CommandRunner interface {
	Run(ctx context.Context, dir, command string, args ...string) (string, error)
}

// ExecRunner implements CommandRunner using os/exec.
type ExecRunner struct {
	Timeout time.Duration
}

// NewExecRunner creates a runner with the given timeout.
func NewExecRunner(timeout time.Duration) *ExecRunner {
	if timeout == 0 {
		timeout = DefaultTimeout
	}
	return &ExecRunner{Timeout: timeout}
}

// Run executes a command with timeout and returns trimmed stdout.
func (r *ExecRunner) Run(ctx context.Context, dir, command string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", &CommandError{
			Command: strings.TrimSpace(command + " " + strings.Join(args, " ")),
			Args:    args,
			Stderr:  strings.TrimSpace(stderr.String()),
			Err:     err,
		}
	}

	return strings.TrimSpace(stdout.String()), nil
}
