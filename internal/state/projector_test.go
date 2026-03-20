package state

import (
	"fmt"
	"testing"
	"time"
)

// mockProjectionStore counts Project calls for testing.
type mockProjectionStore struct {
	projected []Event
	err       error
}

func (m *mockProjectionStore) Project(evt Event) error {
	m.projected = append(m.projected, evt)
	return m.err
}

func (m *mockProjectionStore) GetRequirement(id string) (Requirement, error)         { return Requirement{}, nil }
func (m *mockProjectionStore) GetStory(id string) (Story, error)                     { return Story{}, nil }
func (m *mockProjectionStore) ListRequirements(filter ReqFilter) ([]Requirement, error) { return nil, nil }
func (m *mockProjectionStore) ListStories(filter StoryFilter) ([]Story, error)       { return nil, nil }
func (m *mockProjectionStore) ListAgents(filter AgentFilter) ([]Agent, error)        { return nil, nil }
func (m *mockProjectionStore) ListEscalations() ([]Escalation, error)                { return nil, nil }
func (m *mockProjectionStore) ListStoryDeps(reqID string) ([]StoryDep, error)        { return nil, nil }
func (m *mockProjectionStore) ArchiveRequirement(reqID string) error                 { return nil }
func (m *mockProjectionStore) ArchiveStoriesByReq(reqID string) error                { return nil }
func (m *mockProjectionStore) Close() error                                          { return nil }

func TestProjector_EventsAreProjected(t *testing.T) {
	mock := &mockProjectionStore{}
	p := NewProjector(mock, 10)
	p.Start()

	evt := NewEvent(EventReqSubmitted, "user", "", map[string]any{"id": "r1"})
	p.Send(evt)

	// Give goroutine time to process
	time.Sleep(50 * time.Millisecond)
	p.Shutdown()

	if len(mock.projected) != 1 {
		t.Fatalf("expected 1 projected event, got %d", len(mock.projected))
	}
	if mock.projected[0].ID != evt.ID {
		t.Error("projected event ID mismatch")
	}
}

func TestProjector_ShutdownDrainsChannel(t *testing.T) {
	mock := &mockProjectionStore{}
	p := NewProjector(mock, 100)
	p.Start()

	// Send multiple events quickly
	for i := 0; i < 50; i++ {
		p.Send(NewEvent(EventStoryCreated, "planner", "", map[string]any{}))
	}

	p.Shutdown()

	if len(mock.projected) != 50 {
		t.Fatalf("expected 50 projected events after shutdown, got %d", len(mock.projected))
	}
}

func TestProjector_ErrorsDontCrashGoroutine(t *testing.T) {
	mock := &mockProjectionStore{err: fmt.Errorf("db error")}
	p := NewProjector(mock, 10)
	p.Start()

	// Send events — errors should be logged but not crash
	p.Send(NewEvent(EventReqSubmitted, "user", "", map[string]any{"id": "r1"}))
	p.Send(NewEvent(EventReqSubmitted, "user", "", map[string]any{"id": "r2"}))

	time.Sleep(50 * time.Millisecond)
	p.Shutdown()

	// Both events should have been attempted
	if len(mock.projected) != 2 {
		t.Fatalf("expected 2 projection attempts, got %d", len(mock.projected))
	}
}

func TestProjector_MultipleShutdownsSafe(t *testing.T) {
	mock := &mockProjectionStore{}
	p := NewProjector(mock, 10)
	p.Start()
	p.Shutdown()
	p.Shutdown() // should not panic
}

func TestProjector_SendAfterShutdown(t *testing.T) {
	mock := &mockProjectionStore{}
	p := NewProjector(mock, 10)
	p.Start()
	p.Shutdown()

	// Send after shutdown should not panic (just drop the event)
	p.Send(NewEvent(EventReqSubmitted, "user", "", map[string]any{}))
}
