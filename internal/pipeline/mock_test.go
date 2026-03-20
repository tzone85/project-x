package pipeline

import (
	"context"
	"sync"
)

// mockStage is a test double for the Stage interface.
type mockStage struct {
	name    string
	result  StageResult
	err     error
	calls   int
	mu      sync.Mutex
}

func newMockStage(name string, result StageResult, err error) *mockStage {
	return &mockStage{name: name, result: result, err: err}
}

func (s *mockStage) Name() string { return s.name }

func (s *mockStage) Execute(_ context.Context, _ StoryContext) (StageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.result, s.err
}

func (s *mockStage) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// failThenPassStage fails N times then passes.
type failThenPassStage struct {
	name     string
	failErr  error
	failFor  int
	calls    int
	mu       sync.Mutex
}

func newFailThenPassStage(name string, failFor int, err error) *failThenPassStage {
	return &failThenPassStage{name: name, failFor: failFor, failErr: err}
}

func (s *failThenPassStage) Name() string { return s.name }

func (s *failThenPassStage) Execute(_ context.Context, _ StoryContext) (StageResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	if s.calls <= s.failFor {
		return Failed, s.failErr
	}
	return Passed, nil
}

func (s *failThenPassStage) CallCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

// mockBudgetChecker is a test double for BudgetChecker.
type mockBudgetChecker struct {
	mu  sync.Mutex
	err error
}

func newMockBudget() *mockBudgetChecker {
	return &mockBudgetChecker{}
}

func (b *mockBudgetChecker) CheckBudget(_ context.Context, _, _ string) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.err
}

func (b *mockBudgetChecker) SetError(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.err = err
}

// mockEventEmitter is a test double for EventEmitter.
type mockEventEmitter struct {
	mu       sync.Mutex
	started  []startedEvent
	updated  []updatedEvent
}

type startedEvent struct {
	storyID string
	stage   string
	attempt int
}

type updatedEvent struct {
	storyID    string
	stage      string
	status     string
	attempt    int
	err        string
	durationMs int64
}

func newMockEmitter() *mockEventEmitter {
	return &mockEventEmitter{}
}

func (e *mockEventEmitter) EmitRunStarted(storyID, stage string, attempt int) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.started = append(e.started, startedEvent{storyID, stage, attempt})
	return nil
}

func (e *mockEventEmitter) EmitRunUpdated(storyID, stage, status string, attempt int, stageErr string, durationMs int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.updated = append(e.updated, updatedEvent{storyID, stage, status, attempt, stageErr, durationMs})
	return nil
}

func (e *mockEventEmitter) StartedCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.started)
}

func (e *mockEventEmitter) UpdatedCount() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return len(e.updated)
}

func (e *mockEventEmitter) LastUpdatedStatus() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.updated) == 0 {
		return ""
	}
	return e.updated[len(e.updated)-1].status
}
