package runtime

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/tzone85/project-x/internal/git"
	"github.com/tzone85/project-x/internal/tmux"
)

// Detection patterns for Claude Code output.
var (
	claudePermissionRe = regexp.MustCompile(
		`(?i)(Allow\s+.*\?\s*\(y/n\)|Yes\s*/\s*No|approve\s+this|Do you want to allow)`,
	)
	claudePlanModeRe = regexp.MustCompile(
		`(?i)(plan\s*mode|Plan:\s+)`,
	)
	claudeIdleRe = regexp.MustCompile(
		`(?m)^\$\s*$`,
	)
)

// ClaudeCodeRuntime implements the Runtime interface for the Claude Code CLI.
type ClaudeCodeRuntime struct {
	godmode bool
}

// NewClaudeCodeRuntime creates a ClaudeCodeRuntime.
// When godmode is true, Spawn passes --dangerously-skip-permissions.
func NewClaudeCodeRuntime(godmode bool) *ClaudeCodeRuntime {
	return &ClaudeCodeRuntime{godmode: godmode}
}

// Name returns "claude-code".
func (c *ClaudeCodeRuntime) Name() string {
	return "claude-code"
}

// Version detects the installed Claude CLI version.
func (c *ClaudeCodeRuntime) Version(runner git.CommandRunner) (string, error) {
	out, err := runner.Run("", "claude", "--version")
	if err != nil {
		return "", fmt.Errorf("claude version: %w", err)
	}
	return strings.TrimSpace(out), nil
}

// Health checks the health of a Claude Code session via tmux.
func (c *ClaudeCodeRuntime) Health(runner git.CommandRunner, sessionName string) (tmux.HealthResult, error) {
	return tmux.SessionHealth(runner, sessionName, ""), nil
}

// Spawn starts a Claude Code session inside a new tmux session.
func (c *ClaudeCodeRuntime) Spawn(runner git.CommandRunner, cfg SessionConfig) error {
	cmd := c.buildCommand(cfg)
	return tmux.CreateSession(runner, cfg.SessionName, cfg.WorkDir, cmd)
}

// Kill terminates the tmux session.
func (c *ClaudeCodeRuntime) Kill(runner git.CommandRunner, sessionName string) error {
	return tmux.KillSession(runner, sessionName)
}

// DetectStatus reads pane output and classifies the agent state.
func (c *ClaudeCodeRuntime) DetectStatus(runner git.CommandRunner, sessionName string) (AgentStatus, error) {
	if !tmux.SessionExists(runner, sessionName) {
		return StatusDone, nil
	}

	output, err := tmux.ReadOutput(runner, sessionName, 50)
	if err != nil {
		return StatusWorking, nil
	}

	return c.classifyOutput(output), nil
}

// ReadOutput returns the last N lines from the tmux pane.
func (c *ClaudeCodeRuntime) ReadOutput(runner git.CommandRunner, sessionName string, lines int) (string, error) {
	return tmux.ReadOutput(runner, sessionName, lines)
}

// SendInput sends keystrokes to the tmux session.
func (c *ClaudeCodeRuntime) SendInput(runner git.CommandRunner, sessionName string, input string) error {
	return tmux.SendKeys(runner, sessionName, input)
}

// Capabilities returns what Claude Code supports.
func (c *ClaudeCodeRuntime) Capabilities() RuntimeCapabilities {
	return RuntimeCapabilities{
		SupportsModel: []string{
			"claude-sonnet-4-20250514",
			"claude-opus-4-20250514",
			"claude-haiku-3-5-20241022",
		},
		SupportsGodmode:    c.godmode,
		SupportsLogFile:    true,
		SupportsJsonOutput: true,
		MaxPromptLength:    0,
		CostTier:           CostTierSubscription,
	}
}

// buildCommand constructs the claude CLI invocation string.
// The goal is piped via stdin using a heredoc to avoid shell argument
// length limits and special character issues with long prompts.
func (c *ClaudeCodeRuntime) buildCommand(cfg SessionConfig) string {
	var parts []string
	parts = append(parts, "claude")

	if c.godmode {
		parts = append(parts, "--dangerously-skip-permissions")
	}

	if cfg.Model != "" {
		parts = append(parts, "--model", cfg.Model)
	}

	if cfg.SystemPrompt != "" {
		parts = append(parts, "--system-prompt", shellQuote(cfg.SystemPrompt))
	}

	if cfg.LogFile != "" {
		parts = append(parts, "--output-file", shellQuote(cfg.LogFile))
	}

	// Pipe the goal via stdin using a heredoc to avoid shell arg length limits.
	// The -p - flag tells Claude CLI to read the prompt from stdin.
	parts = append(parts, "-p", "-")
	cmd := strings.Join(parts, " ")

	// Use a heredoc to pipe the goal into stdin.
	// The PX_EOF delimiter is unlikely to appear in prompts.
	return "cat <<'PX_EOF' | " + cmd + "\n" + cfg.Goal + "\nPX_EOF"
}

// classifyOutput matches output against known patterns.
func (c *ClaudeCodeRuntime) classifyOutput(output string) AgentStatus {
	if claudePermissionRe.MatchString(output) {
		return StatusPermissionPrompt
	}
	if claudePlanModeRe.MatchString(output) {
		return StatusPlanMode
	}

	// Check last non-empty line for idle shell prompt.
	trimmed := strings.TrimRight(output, " \t\n")
	lines := strings.Split(trimmed, "\n")
	if len(lines) > 0 {
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		if claudeIdleRe.MatchString(lastLine) {
			return StatusIdle
		}
	}

	return StatusWorking
}

// shellQuote wraps a string in single quotes for safe shell embedding.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
