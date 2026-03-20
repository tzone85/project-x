package runtime

import (
	"regexp"
	"strings"

	"github.com/tzone85/project-x/internal/git"
	"github.com/tzone85/project-x/internal/tmux"
)

// Detection patterns for Codex CLI output.
var (
	codexPermissionRe = regexp.MustCompile(
		`(?i)(confirm\s+action|proceed\?\s*\[y/n\]|allow\s+this)`,
	)
	codexIdleRe = regexp.MustCompile(
		`(?m)^\$\s*$`,
	)
)

// CodexRuntime implements the Runtime interface for the OpenAI Codex CLI.
type CodexRuntime struct{}

// NewCodexRuntime creates a CodexRuntime.
func NewCodexRuntime() *CodexRuntime {
	return &CodexRuntime{}
}

// Name returns "codex".
func (c *CodexRuntime) Name() string {
	return "codex"
}

// Spawn starts a Codex session inside a new tmux session.
func (c *CodexRuntime) Spawn(runner git.CommandRunner, cfg SessionConfig) error {
	cmd := c.buildCommand(cfg)
	return tmux.CreateSession(runner, cfg.SessionName, cfg.WorkDir, cmd)
}

// Kill terminates the tmux session.
func (c *CodexRuntime) Kill(runner git.CommandRunner, sessionName string) error {
	return tmux.KillSession(runner, sessionName)
}

// DetectStatus reads pane output and classifies the agent state.
func (c *CodexRuntime) DetectStatus(runner git.CommandRunner, sessionName string) (AgentStatus, error) {
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
func (c *CodexRuntime) ReadOutput(runner git.CommandRunner, sessionName string, lines int) (string, error) {
	return tmux.ReadOutput(runner, sessionName, lines)
}

// SendInput sends keystrokes to the tmux session.
func (c *CodexRuntime) SendInput(runner git.CommandRunner, sessionName string, input string) error {
	return tmux.SendKeys(runner, sessionName, input)
}

// Capabilities returns what Codex supports.
func (c *CodexRuntime) Capabilities() RuntimeCapabilities {
	return RuntimeCapabilities{
		SupportsModel: []string{
			"gpt-5.4",
			"gpt-5-codex",
			"gpt-5.2-codex",
			"o3",
			"o4-mini",
		},
		SupportsGodmode:    false,
		SupportsLogFile:    false,
		SupportsJsonOutput: false,
		MaxPromptLength:    0,
	}
}

// buildCommand constructs the codex CLI invocation string.
func (c *CodexRuntime) buildCommand(cfg SessionConfig) string {
	var parts []string
	parts = append(parts, "codex")

	if cfg.Model != "" {
		parts = append(parts, "--model", cfg.Model)
	}

	parts = append(parts, shellQuote(cfg.Goal))

	return strings.Join(parts, " ")
}

// classifyOutput matches Codex output against known patterns.
func (c *CodexRuntime) classifyOutput(output string) AgentStatus {
	if codexPermissionRe.MatchString(output) {
		return StatusPermissionPrompt
	}

	trimmed := strings.TrimRight(output, " \t\n")
	lines := strings.Split(trimmed, "\n")
	if len(lines) > 0 {
		lastLine := strings.TrimSpace(lines[len(lines)-1])
		if codexIdleRe.MatchString(lastLine) {
			return StatusIdle
		}
	}

	return StatusWorking
}
