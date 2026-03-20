package monitor

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/tzone85/project-x/internal/state"
)

func newTestPoller(
	checker *mockSessionChecker,
	dispatcher *mockPipelineDispatcher,
	tracker *mockWaveTracker,
	emitter *mockEventEmitter,
) *Poller {
	wc := NewWaveChecker(tracker, slog.Default())
	return NewPoller(
		checker,
		dispatcher,
		wc,
		emitter,
		PollerConfig{PollInterval: 10 * time.Millisecond, StopTimeout: 1 * time.Second},
		slog.Default(),
	)
}

func TestNewPoller(t *testing.T) {
	p := newTestPoller(newMockChecker(), newMockDispatcher(), newMockWaveTracker(), newMockEmitter())
	if p == nil {
		t.Fatal("expected non-nil poller")
	}
	if p.TrackedCount() != 0 {
		t.Errorf("expected 0 tracked agents, got %d", p.TrackedCount())
	}
}

func TestNewPollerNilLogger(t *testing.T) {
	wc := NewWaveChecker(newMockWaveTracker(), nil)
	p := NewPoller(newMockChecker(), newMockDispatcher(), wc, newMockEmitter(),
		DefaultPollerConfig(), nil)
	if p == nil {
		t.Fatal("expected non-nil poller with nil logger")
	}
}

func TestTrackUntrack(t *testing.T) {
	p := newTestPoller(newMockChecker(), newMockDispatcher(), newMockWaveTracker(), newMockEmitter())

	p.Track("agent-1", "story-1", "req-1", "session-1", 1)
	if p.TrackedCount() != 1 {
		t.Fatalf("expected 1 tracked, got %d", p.TrackedCount())
	}

	p.Track("agent-2", "story-2", "req-1", "session-2", 1)
	if p.TrackedCount() != 2 {
		t.Fatalf("expected 2 tracked, got %d", p.TrackedCount())
	}

	p.Untrack("agent-1")
	if p.TrackedCount() != 1 {
		t.Fatalf("expected 1 tracked after untrack, got %d", p.TrackedCount())
	}

	p.Untrack("agent-2")
	if p.TrackedCount() != 0 {
		t.Fatalf("expected 0 tracked, got %d", p.TrackedCount())
	}
}

func TestUntrackNonexistent(t *testing.T) {
	p := newTestPoller(newMockChecker(), newMockDispatcher(), newMockWaveTracker(), newMockEmitter())
	p.Untrack("does-not-exist") // should not panic
	if p.TrackedCount() != 0 {
		t.Fatalf("expected 0 tracked, got %d", p.TrackedCount())
	}
}

func TestPollOnceHealthy(t *testing.T) {
	checker := newMockChecker()
	dispatcher := newMockDispatcher()
	p := newTestPoller(checker, dispatcher, newMockWaveTracker(), newMockEmitter())

	checker.SetResult("agent-1", StatusHealthy)
	p.Track("agent-1", "story-1", "req-1", "session-1", 1)

	p.pollOnce(context.Background())

	if p.TrackedCount() != 1 {
		t.Error("healthy agent should remain tracked")
	}
	if dispatcher.DispatchCount() != 0 {
		t.Error("healthy agent should not trigger dispatch")
	}
}

func TestPollOnceStale(t *testing.T) {
	checker := newMockChecker()
	p := newTestPoller(checker, newMockDispatcher(), newMockWaveTracker(), newMockEmitter())

	checker.SetResult("agent-1", StatusStale)
	p.Track("agent-1", "story-1", "req-1", "session-1", 1)

	p.pollOnce(context.Background())

	if p.TrackedCount() != 1 {
		t.Error("stale agent should remain tracked (watchdog handles recovery)")
	}
}

func TestPollOnceDead(t *testing.T) {
	checker := newMockChecker()
	dispatcher := newMockDispatcher()
	tracker := newMockWaveTracker()
	tracker.AddStory(state.Story{ID: "story-1", ReqID: "req-1", Wave: 1, Status: "in_progress"})
	p := newTestPoller(checker, dispatcher, tracker, newMockEmitter())

	checker.SetResult("agent-1", StatusDead)
	p.Track("agent-1", "story-1", "req-1", "session-1", 1)

	p.pollOnce(context.Background())

	if p.TrackedCount() != 0 {
		t.Error("dead agent should be untracked")
	}
	if dispatcher.DispatchCount() != 1 {
		t.Errorf("expected 1 dispatch, got %d", dispatcher.DispatchCount())
	}
}

func TestPollOnceMissing(t *testing.T) {
	checker := newMockChecker()
	emitter := newMockEmitter()
	p := newTestPoller(checker, newMockDispatcher(), newMockWaveTracker(), emitter)

	checker.SetResult("agent-1", StatusMissing)
	p.Track("agent-1", "story-1", "req-1", "session-1", 1)

	p.pollOnce(context.Background())

	if p.TrackedCount() != 0 {
		t.Error("missing agent should be untracked")
	}
	if emitter.EventCount() != 1 {
		t.Errorf("expected 1 event emitted, got %d", emitter.EventCount())
	}
	if emitter.LastEventType() != state.EventAgentLost {
		t.Errorf("expected agent_lost event, got %s", emitter.LastEventType())
	}
}

func TestPollOnceCheckerError(t *testing.T) {
	checker := newMockChecker()
	checker.SetError(errors.New("connection refused"))
	p := newTestPoller(checker, newMockDispatcher(), newMockWaveTracker(), newMockEmitter())

	p.Track("agent-1", "story-1", "req-1", "session-1", 1)
	p.pollOnce(context.Background())

	if p.TrackedCount() != 1 {
		t.Error("agent should remain tracked on checker error")
	}
}

func TestPollOnceDispatchError(t *testing.T) {
	checker := newMockChecker()
	dispatcher := newMockDispatcher()
	dispatcher.SetError(errors.New("pipeline busy"))
	tracker := newMockWaveTracker()
	tracker.AddStory(state.Story{ID: "story-1", ReqID: "req-1", Wave: 1, Status: "in_progress"})
	p := newTestPoller(checker, dispatcher, tracker, newMockEmitter())

	checker.SetResult("agent-1", StatusDead)
	p.Track("agent-1", "story-1", "req-1", "session-1", 1)

	p.pollOnce(context.Background())

	// Agent should still be untracked despite dispatch error
	if p.TrackedCount() != 0 {
		t.Error("dead agent should be untracked even on dispatch error")
	}
}

func TestPollOnceStoryNotFound(t *testing.T) {
	checker := newMockChecker()
	dispatcher := newMockDispatcher()
	tracker := newMockWaveTracker() // no stories added
	p := newTestPoller(checker, dispatcher, tracker, newMockEmitter())

	checker.SetResult("agent-1", StatusDead)
	p.Track("agent-1", "story-1", "req-1", "session-1", 1)

	p.pollOnce(context.Background())

	if dispatcher.DispatchCount() != 0 {
		t.Error("should not dispatch when story not found")
	}
}

func TestRunCancellation(t *testing.T) {
	p := newTestPoller(newMockChecker(), newMockDispatcher(), newMockWaveTracker(), newMockEmitter())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		p.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// good
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not exit after context cancellation")
	}
}

func TestRunPollsMultipleTimes(t *testing.T) {
	checker := newMockChecker()
	p := newTestPoller(checker, newMockDispatcher(), newMockWaveTracker(), newMockEmitter())

	checker.SetResult("agent-1", StatusHealthy)
	p.Track("agent-1", "story-1", "req-1", "session-1", 1)

	ctx, cancel := context.WithCancel(context.Background())
	go p.Run(ctx)

	// Wait long enough for several polls (10ms interval)
	time.Sleep(60 * time.Millisecond)
	cancel()

	if checker.CallCount() < 2 {
		t.Errorf("expected at least 2 poll calls, got %d", checker.CallCount())
	}
}

func TestDefaultPollerConfig(t *testing.T) {
	cfg := DefaultPollerConfig()
	if cfg.PollInterval != 5*time.Second {
		t.Errorf("expected 5s poll interval, got %v", cfg.PollInterval)
	}
	if cfg.StopTimeout != 30*time.Second {
		t.Errorf("expected 30s stop timeout, got %v", cfg.StopTimeout)
	}
}

func TestMultipleAgentsPollOnce(t *testing.T) {
	checker := newMockChecker()
	dispatcher := newMockDispatcher()
	tracker := newMockWaveTracker()
	tracker.AddStory(state.Story{ID: "story-1", ReqID: "req-1", Wave: 1, Status: "in_progress"})
	tracker.AddStory(state.Story{ID: "story-2", ReqID: "req-1", Wave: 1, Status: "in_progress"})
	emitter := newMockEmitter()
	p := newTestPoller(checker, dispatcher, tracker, emitter)

	checker.SetResult("agent-1", StatusDead)
	checker.SetResult("agent-2", StatusHealthy)
	checker.SetResult("agent-3", StatusMissing)

	p.Track("agent-1", "story-1", "req-1", "session-1", 1)
	p.Track("agent-2", "story-2", "req-1", "session-2", 1)
	p.Track("agent-3", "story-3", "req-1", "session-3", 1)

	p.pollOnce(context.Background())

	// agent-1 dead → untracked + dispatched
	// agent-2 healthy → still tracked
	// agent-3 missing → untracked + agent_lost event
	if p.TrackedCount() != 1 {
		t.Errorf("expected 1 tracked agent, got %d", p.TrackedCount())
	}
	if dispatcher.DispatchCount() != 1 {
		t.Errorf("expected 1 dispatch, got %d", dispatcher.DispatchCount())
	}
	if emitter.EventCount() != 1 {
		t.Errorf("expected 1 event, got %d", emitter.EventCount())
	}
}
