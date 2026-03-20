// Package tmux provides tmux session lifecycle management and health monitoring.
package tmux

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultTimeout = 10 * time.Second

// Runner executes tmux commands. Extracted as an interface for testing.
type Runner interface {
	Run(ctx context.Context, args ...string) (string, error)
}

// CLIRunner implements Runner using the tmux CLI.
type CLIRunner struct{}

// Run executes a tmux command and returns its stdout.
func (r CLIRunner) Run(ctx context.Context, args ...string) (string, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultTimeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, "tmux", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("tmux %s: %s: %w", strings.Join(args, " "), stderr.String(), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Session manages tmux session lifecycle.
type Session struct {
	runner Runner
}

// NewSession creates a new Session manager with the given Runner.
func NewSession(runner Runner) *Session {
	return &Session{runner: runner}
}

// CreateSession creates a new detached tmux session.
func (s *Session) CreateSession(ctx context.Context, name string) error {
	_, err := s.runner.Run(ctx, "new-session", "-d", "-s", name)
	return err
}

// KillSession kills the named tmux session.
func (s *Session) KillSession(ctx context.Context, name string) error {
	_, err := s.runner.Run(ctx, "kill-session", "-t", name)
	return err
}

// SessionExists checks if a named session exists.
func (s *Session) SessionExists(ctx context.Context, name string) (bool, error) {
	_, err := s.runner.Run(ctx, "has-session", "-t", name)
	if err != nil {
		if strings.Contains(err.Error(), "no server running") ||
			strings.Contains(err.Error(), "session not found") ||
			strings.Contains(err.Error(), "can't find session") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// ListSessions returns the names of all tmux sessions.
func (s *Session) ListSessions(ctx context.Context) ([]string, error) {
	out, err := s.runner.Run(ctx, "list-sessions", "-F", "#{session_name}")
	if err != nil {
		if strings.Contains(err.Error(), "no server running") {
			return nil, nil
		}
		return nil, err
	}
	if out == "" {
		return nil, nil
	}
	return strings.Split(out, "\n"), nil
}

// SendKeys sends keystrokes to the named session.
func (s *Session) SendKeys(ctx context.Context, name, keys string) error {
	_, err := s.runner.Run(ctx, "send-keys", "-t", name, keys, "Enter")
	return err
}

// CaptureOutput captures the visible pane content of the named session.
func (s *Session) CaptureOutput(ctx context.Context, name string) (string, error) {
	return s.runner.Run(ctx, "capture-pane", "-t", name, "-p")
}

// AttachSession attaches to a tmux session (typically used for interactive use).
func (s *Session) AttachSession(ctx context.Context, name string) error {
	_, err := s.runner.Run(ctx, "attach-session", "-t", name)
	return err
}
