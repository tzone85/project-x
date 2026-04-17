package runtime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tzone85/project-x/internal/git"
)

func TestGeminiRuntime_Name(t *testing.T) {
	rt := NewGeminiRuntime()
	if rt.Name() != "gemini" {
		t.Errorf("expected 'gemini', got %s", rt.Name())
	}
}

func TestGeminiRuntime_Capabilities_Models(t *testing.T) {
	rt := NewGeminiRuntime()
	caps := rt.Capabilities()

	if len(caps.SupportsModel) == 0 {
		t.Error("expected at least one supported model")
	}
	if caps.SupportsGodmode {
		t.Error("gemini should not support godmode")
	}
}

func TestGeminiRuntime_Spawn(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", fmt.Errorf("no session"))
	mock.AddResponse("", nil)

	rt := NewGeminiRuntime()
	cfg := SessionConfig{
		SessionName: "px-gemini-1",
		WorkDir:     "/tmp/work",
		Model:       "gemini-2.5-pro",
		Goal:        "implement feature Y",
	}

	err := rt.Spawn(mock, cfg)
	if err != nil {
		t.Fatalf("spawn: %v", err)
	}

	newCmd := mock.Commands[1]
	lastArg := newCmd.Args[len(newCmd.Args)-1]
	if !strings.Contains(lastArg, "gemini") {
		t.Errorf("expected gemini in command, got %q", lastArg)
	}
	if !strings.Contains(lastArg, "--model") {
		t.Errorf("expected --model flag, got %q", lastArg)
	}
}

func TestGeminiRuntime_Kill(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", nil)

	rt := NewGeminiRuntime()
	err := rt.Kill(mock, "px-gemini-1")
	if err != nil {
		t.Fatalf("kill: %v", err)
	}
}

func TestGeminiRuntime_ReadOutput(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("gemini output here", nil)

	rt := NewGeminiRuntime()
	out, err := rt.ReadOutput(mock, "px-gemini-1", 30)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if out != "gemini output here" {
		t.Errorf("expected 'gemini output here', got %q", out)
	}
}

func TestGeminiRuntime_SendInput(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", nil)

	rt := NewGeminiRuntime()
	err := rt.SendInput(mock, "px-gemini-1", "y")
	if err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestGeminiRuntime_DetectStatus_Working(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", nil)
	mock.AddResponse("Generating response...", nil)

	rt := NewGeminiRuntime()
	status, err := rt.DetectStatus(mock, "px-gemini-1")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if status != StatusWorking {
		t.Errorf("expected StatusWorking, got %s", status)
	}
}

func TestGeminiRuntime_DetectStatus_Done(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", fmt.Errorf("no session"))

	rt := NewGeminiRuntime()
	status, err := rt.DetectStatus(mock, "px-gemini-1")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if status != StatusDone {
		t.Errorf("expected StatusDone, got %s", status)
	}
}

func TestGeminiRuntime_DetectStatus_PermissionPrompt(t *testing.T) {
	tests := []struct {
		name   string
		output string
	}{
		{"approve_action", "Approve action: write file?"},
		{"allow_yn", "Allow? (y/n)"},
		{"confirm_execution", "Please confirm execution of command"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mock := git.NewMockRunner()
			mock.AddResponse("", nil)
			mock.AddResponse(tc.output, nil)

			rt := NewGeminiRuntime()
			status, err := rt.DetectStatus(mock, "px-gemini-1")
			if err != nil {
				t.Fatalf("detect: %v", err)
			}
			if status != StatusPermissionPrompt {
				t.Errorf("expected StatusPermissionPrompt for %q, got %s", tc.name, status)
			}
		})
	}
}

func TestGeminiRuntime_DetectStatus_Idle(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", nil)
	mock.AddResponse("output done\n$", nil)

	rt := NewGeminiRuntime()
	status, err := rt.DetectStatus(mock, "px-gemini-1")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if status != StatusIdle {
		t.Errorf("expected StatusIdle, got %s", status)
	}
}

func TestGeminiRuntime_DetectStatus_ReadError(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", nil)
	mock.AddResponse("", fmt.Errorf("capture failed"))

	rt := NewGeminiRuntime()
	status, err := rt.DetectStatus(mock, "px-gemini-1")
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if status != StatusWorking {
		t.Errorf("expected StatusWorking on read error, got %s", status)
	}
}

func TestGeminiRuntime_BuildCommand_NoModel(t *testing.T) {
	rt := NewGeminiRuntime()
	cmd := rt.buildCommand(SessionConfig{Goal: "do something"})
	if !strings.HasPrefix(cmd, "gemini") {
		t.Errorf("expected command to start with 'gemini', got %q", cmd)
	}
	if strings.Contains(cmd, "--model") {
		t.Error("expected no --model flag when model is empty")
	}
}

func TestGeminiRuntime_VersionError(t *testing.T) {
	mock := git.NewMockRunner()
	mock.AddResponse("", fmt.Errorf("not found"))

	rt := NewGeminiRuntime()
	_, err := rt.Version(mock)
	if err == nil {
		t.Error("expected error when gemini not found")
	}
}
