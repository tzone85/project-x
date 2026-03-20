package git

import (
	"context"
	"testing"
	"time"
)

func TestExecRunner_Run_Success(t *testing.T) {
	runner := NewExecRunner(5 * time.Second)
	out, err := runner.Run(context.Background(), "", "echo", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello" {
		t.Errorf("expected 'hello', got %q", out)
	}
}

func TestExecRunner_Run_TrimsWhitespace(t *testing.T) {
	runner := NewExecRunner(5 * time.Second)
	out, err := runner.Run(context.Background(), "", "printf", "  hello  \n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out != "hello" {
		t.Errorf("expected 'hello', got %q", out)
	}
}

func TestExecRunner_Run_Error(t *testing.T) {
	runner := NewExecRunner(5 * time.Second)
	_, err := runner.Run(context.Background(), "", "false")
	if err == nil {
		t.Fatal("expected error")
	}
	cmdErr, ok := err.(*CommandError)
	if !ok {
		t.Fatalf("expected *CommandError, got %T", err)
	}
	if cmdErr.Command != "false" {
		t.Errorf("expected command 'false', got %q", cmdErr.Command)
	}
}

func TestExecRunner_Run_Timeout(t *testing.T) {
	runner := NewExecRunner(100 * time.Millisecond)
	_, err := runner.Run(context.Background(), "", "sleep", "10")
	if err == nil {
		t.Fatal("expected timeout error")
	}
}

func TestExecRunner_Run_WithDir(t *testing.T) {
	dir := t.TempDir()
	runner := NewExecRunner(5 * time.Second)
	out, err := runner.Run(context.Background(), dir, "pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out == "" {
		t.Error("expected non-empty output")
	}
}

func TestExecRunner_Run_Stderr(t *testing.T) {
	runner := NewExecRunner(5 * time.Second)
	_, err := runner.Run(context.Background(), "", "sh", "-c", "echo error >&2; exit 1")
	if err == nil {
		t.Fatal("expected error")
	}
	cmdErr := err.(*CommandError)
	if cmdErr.Stderr != "error" {
		t.Errorf("expected stderr 'error', got %q", cmdErr.Stderr)
	}
}

func TestNewExecRunner_DefaultTimeout(t *testing.T) {
	runner := NewExecRunner(0)
	if runner.Timeout != DefaultTimeout {
		t.Errorf("expected default timeout %v, got %v", DefaultTimeout, runner.Timeout)
	}
}
