package tmux

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// mockRunner is a test double for Runner.
type mockRunner struct {
	responses map[string]string // key: first arg → response
	errors    map[string]error
	calls     [][]string
}

func newMockRunner() *mockRunner {
	return &mockRunner{
		responses: make(map[string]string),
		errors:    make(map[string]error),
	}
}

func (m *mockRunner) Run(_ context.Context, args ...string) (string, error) {
	m.calls = append(m.calls, args)
	key := args[0]
	if err, ok := m.errors[key]; ok {
		return "", err
	}
	if resp, ok := m.responses[key]; ok {
		return resp, nil
	}
	return "", nil
}

// --- Session tests ---

func TestCreateSession(t *testing.T) {
	mr := newMockRunner()
	s := NewSession(mr)

	err := s.CreateSession(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mr.calls) != 1 || mr.calls[0][0] != "new-session" {
		t.Errorf("expected new-session call, got %v", mr.calls)
	}
}

func TestKillSession(t *testing.T) {
	mr := newMockRunner()
	s := NewSession(mr)

	err := s.KillSession(context.Background(), "test-session")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mr.calls[0][0] != "kill-session" {
		t.Errorf("expected kill-session call")
	}
}

func TestSessionExistsTrue(t *testing.T) {
	mr := newMockRunner()
	s := NewSession(mr)

	exists, err := s.SessionExists(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected session to exist")
	}
}

func TestSessionExistsFalse(t *testing.T) {
	mr := newMockRunner()
	mr.errors["has-session"] = fmt.Errorf("can't find session: test")
	s := NewSession(mr)

	exists, err := s.SessionExists(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected session to not exist")
	}
}

func TestSessionExistsNoServer(t *testing.T) {
	mr := newMockRunner()
	mr.errors["has-session"] = fmt.Errorf("no server running on /tmp/tmux-1000/default")
	s := NewSession(mr)

	exists, err := s.SessionExists(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected session to not exist when no server")
	}
}

func TestListSessions(t *testing.T) {
	mr := newMockRunner()
	mr.responses["list-sessions"] = "session1\nsession2\nsession3"
	s := NewSession(mr)

	sessions, err := s.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sessions) != 3 {
		t.Errorf("got %d sessions, want 3", len(sessions))
	}
}

func TestListSessionsEmpty(t *testing.T) {
	mr := newMockRunner()
	mr.errors["list-sessions"] = fmt.Errorf("no server running")
	s := NewSession(mr)

	sessions, err := s.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessions != nil {
		t.Errorf("expected nil, got %v", sessions)
	}
}

func TestSendKeys(t *testing.T) {
	mr := newMockRunner()
	s := NewSession(mr)

	err := s.SendKeys(context.Background(), "test", "echo hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if mr.calls[0][0] != "send-keys" {
		t.Errorf("expected send-keys call")
	}
}

func TestCaptureOutput(t *testing.T) {
	mr := newMockRunner()
	mr.responses["capture-pane"] = "line1\nline2\nline3"
	s := NewSession(mr)

	output, err := s.CaptureOutput(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if output != "line1\nline2\nline3" {
		t.Errorf("output = %q", output)
	}
}

func TestAttachSession(t *testing.T) {
	mr := newMockRunner()
	s := NewSession(mr)

	err := s.AttachSession(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Health tests ---

func TestHealthMissing(t *testing.T) {
	mr := newMockRunner()
	mr.errors["has-session"] = fmt.Errorf("can't find session")
	s := NewSession(mr)
	hm := NewHealthMonitor(s, DefaultHealthConfig())

	result := hm.SessionHealth(context.Background(), "missing")
	if result.Status != HealthMissing {
		t.Errorf("status = %q, want missing", result.Status)
	}
}

func TestHealthDead(t *testing.T) {
	mr := newMockRunner()
	mr.responses["list-panes"] = "12345 1 1"
	s := NewSession(mr)
	hm := NewHealthMonitor(s, DefaultHealthConfig())

	result := hm.SessionHealth(context.Background(), "dead-session")
	if result.Status != HealthDead {
		t.Errorf("status = %q, want dead", result.Status)
	}
	if result.ExitCode != "1" {
		t.Errorf("exit code = %q, want 1", result.ExitCode)
	}
}

func TestHealthDeadNoExitCode(t *testing.T) {
	mr := newMockRunner()
	mr.responses["list-panes"] = "12345 1"
	s := NewSession(mr)
	hm := NewHealthMonitor(s, DefaultHealthConfig())

	result := hm.SessionHealth(context.Background(), "dead-session")
	if result.Status != HealthDead {
		t.Errorf("status = %q, want dead", result.Status)
	}
}

func TestHealthHealthyOutputChanging(t *testing.T) {
	mr := newMockRunner()
	mr.responses["list-panes"] = "12345 0"
	mr.responses["capture-pane"] = "output-1"
	s := NewSession(mr)
	hm := NewHealthMonitor(s, DefaultHealthConfig())

	result := hm.SessionHealth(context.Background(), "healthy")
	if result.Status != HealthHealthy {
		t.Errorf("status = %q, want healthy", result.Status)
	}

	// Change output
	mr.responses["capture-pane"] = "output-2"
	result = hm.SessionHealth(context.Background(), "healthy")
	if result.Status != HealthHealthy {
		t.Errorf("status = %q, want healthy after output change", result.Status)
	}
}

func TestHealthStaleOutputStatic(t *testing.T) {
	mr := newMockRunner()
	mr.responses["list-panes"] = "12345 0"
	mr.responses["capture-pane"] = "static-output"
	s := NewSession(mr)
	config := HealthConfig{StaleThreshold: 1 * time.Millisecond}
	hm := NewHealthMonitor(s, config)

	// First check — establishes baseline
	hm.SessionHealth(context.Background(), "stale")

	// Wait past threshold
	time.Sleep(5 * time.Millisecond)

	// Second check — same output, past threshold → stale
	result := hm.SessionHealth(context.Background(), "stale")
	if result.Status != HealthStale {
		t.Errorf("status = %q, want stale", result.Status)
	}
}

func TestHealthPaneInfoError(t *testing.T) {
	mr := newMockRunner()
	mr.errors["list-panes"] = fmt.Errorf("pane error")
	s := NewSession(mr)
	hm := NewHealthMonitor(s, DefaultHealthConfig())

	result := hm.SessionHealth(context.Background(), "error-session")
	if result.Status != HealthDead {
		t.Errorf("status = %q, want dead on pane info error", result.Status)
	}
}

func TestHealthCaptureError(t *testing.T) {
	mr := newMockRunner()
	mr.responses["list-panes"] = "12345 0"
	mr.errors["capture-pane"] = fmt.Errorf("capture error")
	s := NewSession(mr)
	hm := NewHealthMonitor(s, DefaultHealthConfig())

	result := hm.SessionHealth(context.Background(), "capture-fail")
	if result.Status != HealthStale {
		t.Errorf("status = %q, want stale on capture error", result.Status)
	}
}

// --- Recovery tracker tests ---

func TestRecoveryTracker(t *testing.T) {
	rt := NewRecoveryTracker(2)

	if !rt.CanRecover("agent-1") {
		t.Error("should be able to recover with 0 attempts")
	}

	rt.RecordAttempt("agent-1")
	if !rt.CanRecover("agent-1") {
		t.Error("should be able to recover with 1 attempt")
	}

	rt.RecordAttempt("agent-1")
	if rt.CanRecover("agent-1") {
		t.Error("should not be able to recover after max attempts")
	}
}

func TestRecoveryTrackerReset(t *testing.T) {
	rt := NewRecoveryTracker(1)
	rt.RecordAttempt("agent-1")

	if rt.CanRecover("agent-1") {
		t.Error("should be at max")
	}

	rt.Reset("agent-1")
	if !rt.CanRecover("agent-1") {
		t.Error("should be able to recover after reset")
	}
}

func TestDefaultHealthConfig(t *testing.T) {
	cfg := DefaultHealthConfig()
	if cfg.StaleThreshold != 180*time.Second {
		t.Errorf("stale threshold = %v, want 180s", cfg.StaleThreshold)
	}
	if cfg.MaxRecoveryAttempts != 2 {
		t.Errorf("max recovery = %d, want 2", cfg.MaxRecoveryAttempts)
	}
}

func TestListSessionsEmptyResponse(t *testing.T) {
	mr := newMockRunner()
	mr.responses["list-sessions"] = ""
	s := NewSession(mr)

	sessions, err := s.ListSessions(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sessions != nil {
		t.Errorf("expected nil for empty response, got %v", sessions)
	}
}
