package monitor

import (
	"context"
	"log/slog"
	"testing"

	"github.com/tzone85/project-x/internal/state"
)

func TestWaveChecker_CheckCompletion_AllDone(t *testing.T) {
	tracker := newMockWaveTracker()
	tracker.AddStory(state.Story{ID: "s1", ReqID: "req-1", Wave: 1, Status: "merged"})
	tracker.AddStory(state.Story{ID: "s2", ReqID: "req-1", Wave: 1, Status: "done"})
	tracker.AddStory(state.Story{ID: "s3", ReqID: "req-1", Wave: 2, Status: "planned"})

	wc := NewWaveChecker(tracker, slog.Default())

	if !wc.CheckCompletion("req-1", 1) {
		t.Error("expected wave 1 to be complete")
	}
}

func TestWaveChecker_CheckCompletion_NotDone(t *testing.T) {
	tracker := newMockWaveTracker()
	tracker.AddStory(state.Story{ID: "s1", ReqID: "req-1", Wave: 1, Status: "merged"})
	tracker.AddStory(state.Story{ID: "s2", ReqID: "req-1", Wave: 1, Status: "in_progress"})

	wc := NewWaveChecker(tracker, slog.Default())

	if wc.CheckCompletion("req-1", 1) {
		t.Error("expected wave 1 to be incomplete")
	}
}

func TestWaveChecker_CheckCompletion_EmptyReq(t *testing.T) {
	tracker := newMockWaveTracker()
	wc := NewWaveChecker(tracker, slog.Default())

	// No stories for this req — vacuously true
	if !wc.CheckCompletion("req-nonexistent", 1) {
		t.Error("expected empty wave to be considered complete")
	}
}

func TestWaveChecker_NextWaveStories(t *testing.T) {
	tracker := newMockWaveTracker()
	tracker.AddStory(state.Story{ID: "s1", ReqID: "req-1", Wave: 1, Status: "merged"})
	tracker.AddStory(state.Story{ID: "s2", ReqID: "req-1", Wave: 2, Status: "planned"})
	tracker.AddStory(state.Story{ID: "s3", ReqID: "req-1", Wave: 2, Status: "planned"})
	tracker.AddStory(state.Story{ID: "s4", ReqID: "req-1", Wave: 2, Status: "in_progress"})
	tracker.AddStory(state.Story{ID: "s5", ReqID: "req-1", Wave: 3, Status: "planned"})

	wc := NewWaveChecker(tracker, slog.Default())

	ready := wc.NextWaveStories("req-1", 1)
	if len(ready) != 2 {
		t.Errorf("expected 2 ready stories in wave 2, got %d", len(ready))
	}

	for _, s := range ready {
		if s.Wave != 2 {
			t.Errorf("expected wave 2, got %d", s.Wave)
		}
		if s.Status != "planned" {
			t.Errorf("expected planned status, got %s", s.Status)
		}
	}
}

func TestWaveChecker_NextWaveStories_NoNext(t *testing.T) {
	tracker := newMockWaveTracker()
	tracker.AddStory(state.Story{ID: "s1", ReqID: "req-1", Wave: 1, Status: "merged"})

	wc := NewWaveChecker(tracker, slog.Default())

	ready := wc.NextWaveStories("req-1", 1)
	if len(ready) != 0 {
		t.Errorf("expected 0 ready stories, got %d", len(ready))
	}
}

func TestWaveChecker_NilLogger(t *testing.T) {
	wc := NewWaveChecker(newMockWaveTracker(), nil)
	if wc == nil {
		t.Fatal("expected non-nil WaveChecker with nil logger")
	}
}

func TestWaveCompletionTriggersPipeline(t *testing.T) {
	checker := newMockChecker()
	dispatcher := newMockDispatcher()
	tracker := newMockWaveTracker()
	emitter := newMockEmitter()

	// Wave 1: two stories, both will finish
	tracker.AddStory(state.Story{ID: "s1", ReqID: "req-1", Wave: 1, Status: "merged"})
	tracker.AddStory(state.Story{ID: "s2", ReqID: "req-1", Wave: 1, Status: "in_progress"})
	// Wave 2: one story ready
	tracker.AddStory(state.Story{ID: "s3", ReqID: "req-1", Wave: 2, Status: "planned"})

	p := newTestPoller(checker, dispatcher, tracker, emitter)

	// s2's agent dies — its story should be dispatched to pipeline
	checker.SetResult("agent-2", StatusDead)
	p.Track("agent-2", "s2", "req-1", "session-2", 1)

	p.pollOnce(context.Background())

	if dispatcher.DispatchCount() != 1 {
		t.Errorf("expected 1 dispatch, got %d", dispatcher.DispatchCount())
	}
	// Wave 1 is NOT yet complete because s2 is still "in_progress" in the tracker
	// (the story status gets updated by the pipeline, not by the poller)
}
