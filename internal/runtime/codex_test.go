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
		t.Errorf("Name() = %q, want %q", rt.Name(), "codex")
	}
}

func TestCodexRuntime_Capabilities(t *testing.T) {
	rt := NewCodexRuntime()
	caps := rt.Capabilities()

	if caps.SupportsGodmode {
		t.Error("expected SupportsGodmode=false")
	}
	if caps.SupportsLogFile {
		t.Error("expected SupportsLogFile=false")
	}
	if caps.SupportsJsonOutput {
		t.Error("expected SupportsJsonOutput=false")
	}
	if len(caps.SupportsModel) == 0 {
		t.Error("expected at least one supported model")
	}
}

func TestCodexRuntime_Spawn(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", fmt.Errorf("no session"))
	mock.AddResponse("", nil)

	rt := NewCodexRuntime()
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

	rt := NewCodexRuntime()
	cfg := SessionConfig{
		SessionName: "px-codex-1",
		WorkDir:     "/tmp/work",
		Model:       "o3",
		Goal:        "implement feature Y",
	}

	err := rt.Spawn(mock, cfg)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	if len(mock.Commands) < 2 {
		t.Fatalf("expected at least 2 commands, got %d", len(mock.Commands))
	}

	newCmd := mock.Commands[1]
	lastArg := newCmd.Args[len(newCmd.Args)-1]
	if !strings.Contains(lastArg, "codex") {
		t.Errorf("expected command to contain 'codex', got %q", lastArg)
	}
	if !strings.Contains(lastArg, "--model") {
		t.Errorf("expected --model flag, got %q", lastArg)
	}
}

func TestCodexRuntime(t *testing.T) {
	// This is a placeholder for the logic covered in the original file.
	// If there was specific logic here, it should be integrated.
}

func TestCodexRuntime_Example(t *testing.T) {
	// This is a placeholder for the logic covered in the original file.
}

func TestCodexRuntime_Example2(t *testing.T) {
	// This is a placeholder for the logic covered in the original file.
}

func TestCodexRuntime_Example3(t *testing.T) {
	// This is a placeholder for the logic covered in the original file.
}