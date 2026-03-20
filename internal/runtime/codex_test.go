package runtime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tzone85/project-x/internal/git"
)

func TestCodexRuntime_Name(t *testing.T) {
	rt := NewCodexRuntime(false)
	if rt.Name() != "codex" {
		t.Errorf("expected 'codex', got %s", rt.Name())
	}
}

func TestCodexRuntime_Spawn(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", fmt.Errorf("no session"))
	mock.AddResponse("", nil)

	rt := NewCodexRuntime(false)
	cfg := SessionConfig{
		SessionName: "px-story-1",
		WorkDir:     "/tmp/work",
		Model:       "gpt-5.4",
		Goal:        "implement feature X",
	}

	if err := rt.Spawn(mock, cfg); err != nil {
		t.Fatalf("spawn: %v", err)
	}

	newCmd := mock.Commands[1]
	lastArg := newCmd.Args[len(newCmd.Args)-1]
	if !strings.Contains(lastArg, "codex exec") {
		t.Errorf("expected command to contain 'codex exec', got %q", lastArg)
	}
	if !strings.Contains(lastArg, "--model") {
		t.Errorf("expected command to contain '--model', got %q", lastArg)
	}
	if !strings.Contains(lastArg, "--full-auto") {
		t.Errorf("expected command to contain '--full-auto', got %q", lastArg)
	}
	if !strings.Contains(lastArg, "--sandbox workspace-write") {
		t.Errorf("expected command to contain workspace-write sandbox, got %q", lastArg)
	}
	if !strings.Contains(lastArg, "printf '$") {
		t.Errorf("expected completion marker in command, got %q", lastArg)
	}
}

func TestCodexRuntime_SpawnWithGodmode(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", fmt.Errorf("no session"))
	mock.AddResponse("", nil)

	rt := NewCodexRuntime(true)
	cfg := SessionConfig{
		SessionName: "px-story-1",
		WorkDir:     "/tmp/work",
		Goal:        "implement feature X",
	}

	if err := rt.Spawn(mock, cfg); err != nil {
		t.Fatalf("spawn: %v", err)
	}

	newCmd := mock.Commands[1]
	lastArg := newCmd.Args[len(newCmd.Args)-1]
	if !strings.Contains(lastArg, "--dangerously-bypass-approvals-and-sandbox") {
		t.Errorf("expected Codex godmode flag, got %q", lastArg)
	}
	if strings.Contains(lastArg, "--full-auto") {
		t.Errorf("did not expect --full-auto when godmode is enabled, got %q", lastArg)
	}
}

func TestCodexRuntime_DetectStatus_TrustPrompt(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", nil)
	mock.AddResponse("Do you trust the contents of this directory?\nPress enter to continue", nil)

	rt := NewCodexRuntime(false)
	status, err := rt.DetectStatus(mock, "px-story-1")
	if err != nil {
		t.Fatalf("detect status: %v", err)
	}
	if status != StatusPermissionPrompt {
		t.Fatalf("expected permission prompt, got %s", status)
	}
}
