package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

// --- Root command tests ---

func TestRootCmdExists(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.Use != "px" {
		t.Errorf("root use = %q, want px", cmd.Use)
	}
}

func TestVersionCmd(t *testing.T) {
	SetVersion("1.2.3")

	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "1.2.3") {
		t.Errorf("output = %q, want version 1.2.3", buf.String())
	}
}

func TestPlanCmdNoArgs(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"plan"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for plan without args")
	}
}

func TestPlanCmdWithFile(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"plan", "requirements.md"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(buf.String(), "requirements.md") {
		t.Errorf("output = %q, want file reference", buf.String())
	}
}

func TestResumeCmdNoArgs(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"resume"})

	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for resume without args")
	}
}

func TestResumeCmdWithID(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"resume", "REQ-001"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCostCmd(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"cost"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDashboardCmdTUI(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"dashboard"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDashboardCmdWeb(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"dashboard", "--web", "--port", "8080"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMigrateCmd(t *testing.T) {
	cmd := NewRootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"migrate"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerboseFlag(t *testing.T) {
	cmd := NewRootCmd()
	cmd.SetArgs([]string{"--verbose", "version"})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Shutdown tests ---

func TestShutdownManagerRunsHooks(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	sm := NewShutdownManager(cancel, nil)

	var order []string
	sm.Register(ShutdownHook{
		Name: "hook1",
		Fn: func(_ context.Context) error {
			order = append(order, "hook1")
			return nil
		},
	})
	sm.Register(ShutdownHook{
		Name: "hook2",
		Fn: func(_ context.Context) error {
			order = append(order, "hook2")
			return nil
		},
	})

	sm.Shutdown()

	if len(order) != 2 {
		t.Fatalf("expected 2 hooks, got %d", len(order))
	}
	if order[0] != "hook1" || order[1] != "hook2" {
		t.Errorf("hooks ran out of order: %v", order)
	}
}

func TestShutdownManagerHookError(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	sm := NewShutdownManager(cancel, nil)

	var ranSecond bool
	sm.Register(ShutdownHook{
		Name: "failing",
		Fn: func(_ context.Context) error {
			return context.DeadlineExceeded
		},
	})
	sm.Register(ShutdownHook{
		Name: "second",
		Fn: func(_ context.Context) error {
			ranSecond = true
			return nil
		},
	})

	sm.Shutdown()

	// Second hook should still run after first fails
	if !ranSecond {
		t.Error("second hook should run even after first fails")
	}
}

func TestShutdownManagerCancelsContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sm := NewShutdownManager(cancel, nil)

	sm.Shutdown()

	if ctx.Err() == nil {
		t.Error("context should be cancelled after shutdown")
	}
}

func TestShutdownManagerNoHooks(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	sm := NewShutdownManager(cancel, nil)

	// Should not panic
	sm.Shutdown()
}
