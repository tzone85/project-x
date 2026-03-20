package runtime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tzone85/project-x/internal/git"
)

func TestCodexRuntime_Name(t *testing.T) {
	rt := NewCodexRuntime()
	if rt.Name() != "codex" {
		t.Errorf("expected 'codex', got %s", rt.Name())
	}
}

func TestCodexRuntime_Capabilities_Models(t *testing.T) {
	rt := NewCodexRuntime()
	caps := rt.Capabilities()

	if len(caps.SupportsModel) == 0 {
		t.Error("expected at least one supported model")
	}
	if caps.SupportsGodmode {
		t.Error("codex should not support godmode")
	}
}

func TestCodexRuntime_Spawn(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", fmt.Errorf("no session")) // has-session fails
	mock.AddResponse("", nil)                      // new-session succeeds

	rt := NewCodexRuntime()
	cfg := SessionConfig{
		SessionName: "px-codex-1",
		WorkDir:     "/tmp/work",
		Model:       "gpt-5.4",
		Goal:        "implement feature X",
	}

	err := rt.Spawn(mock, cfg)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	if len(mock.Commands) < 2 {
		t.Fatalf("expected at least 2 commands, got %d", len(mock.Commands))
	}

	newCmd := mock.Commands[1]
	if newCmd.Name != "tmux" {
		t.Errorf("expected tmux, got %s", newCmd.Name)
	}

	lastArg := newCmd.Args[len(newCmd.Args)-1]
	if !strings.Contains(lastArg, "codex") {
		t.Errorf("expected codex in command, got %q", lastArg)
	}
	if !strings.Contains(lastArg, "--model") {
		t.Errorf("expected --model flag, got %q", lastArg)
	}
}

func TestCodexRuntime_Kill(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", nil)

	rt := NewCodexRuntime()
	err := rt.Kill(mock, "px-codex-1")
	if err != nil {
		t.Fatalf("kill: %v", err)
	}

	cmd := mock.Commands[0]
	if cmd.Name != "tmux" {
		t.Errorf("expected tmux, got %s", cmd.Name)
	}
}

func TestCodexRuntime_ReadOutput(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("codex output here", nil)

	rt := NewCodexRuntime()
	out, err := rt.ReadOutput(mock, "px-codex-1", 30)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if out != "codex output here" {
		t.Errorf("expected 'codex output here', got %q", out)
	}
}

func TestCodexRuntime_SendInput(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", nil)

	rt := NewCodexRuntime()
	err := rt.SendInput(mock, "px-codex-1", "y")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestCodexRuntime_DetectStatus_Working(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", nil)                                  // has-session
	mock.AddResponse("Processing request... generating...", nil) // capture-pane

	rt := NewCodexRuntime()
	status, err := rt.DetectStatus(mock, "px-codex-1")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if status != StatusWorking {
		t.Errorf("expected StatusWorking, got %s", status)
	}
}

func TestCodexRuntime_DetectStatus_Done(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", fmt.Errorf("no session"))

	rt := NewCodexRuntime()
	status, err := rt.DetectStatus(mock, "px-codex-1")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if status != StatusDone {
		t.Errorf("expected StatusDone, got %s", status)
	}
}

func TestCodexRuntime_DetectStatus_PermissionPrompt(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{"confirm_action", "Some output\nConfirm action?"},
		{"proceed_yn", "Do you want to proceed? [y/n]"},
		{"allow_this", "Please allow this operation"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := git.NewMockRunner()
			mock.AddResponse("", nil)       // has-session
			mock.AddResponse(tc.output, nil) // capture-pane

			rt := NewCodexRuntime()
			status, err := rt.DetectStatus(mock, "px-codex-1")
			if err != nil {
				t.Fatalf("detect: %v", err)
			}
			if status != StatusPermissionPrompt {
				t.Errorf("expected StatusPermissionPrompt for %q, got %s", tc.name, status)
			}
		})
	}
}

func TestCodexRuntime_DetectStatus_Idle(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", nil)        // has-session
	mock.AddResponse("done\n$", nil) // idle prompt

	rt := NewCodexRuntime()
	status, err := rt.DetectStatus(mock, "px-codex-1")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if status != StatusIdle {
		t.Errorf("expected StatusIdle, got %s", status)
	}
}

func TestCodexRuntime_DetectStatus_ReadError(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", nil)                          // has-session
	mock.AddResponse("", fmt.Errorf("capture failed")) // capture-pane fails

	rt := NewCodexRuntime()
	status, err := rt.DetectStatus(mock, "px-codex-1")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if status != StatusWorking {
		t.Errorf("expected StatusWorking on read error, got %s", status)
	}
}

func TestCodexRuntime_BuildCommand_NoModel(t *testing.T) {
	rt := NewCodexRuntime()
	cmd := rt.buildCommand(SessionConfig{Goal: "do something"})
	if !strings.HasPrefix(cmd, "codex") {
		t.Errorf("expected command to start with 'codex', got %q", cmd)
	}
	if strings.Contains(cmd, "--model") {
		t.Error("expected no --model flag when model is empty")
	}
}

func TestCodexRuntime_VersionError(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", fmt.Errorf("not found"))

	rt := NewCodexRuntime()
	_, err := rt.Version(mock)
	if err == nil {
		t.Error("expected error when codex not found")
	}
}
