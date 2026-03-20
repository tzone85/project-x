package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEventStore_AppendAndReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	es, err := NewEventStore(path)
	if err != nil {
		t.Fatalf("NewEventStore: %v", err)
	}

	// Append events
	events := []struct {
		typ     EventType
		payload any
	}{
		{EventRequirementCreated, RequirementCreatedPayload{
			RequirementID: "req-1", Title: "Req 1", Description: "Desc", Source: "test",
		}},
		{EventStoryCreated, StoryCreatedPayload{
			StoryID: "story-1", RequirementID: "req-1", Title: "Story 1",
			Complexity: 3, OwnedFiles: []string{"a.go"}, DependsOn: []string{},
		}},
		{EventStoryStatusChanged, StoryStatusChangedPayload{
			StoryID: "story-1", OldStatus: "planned", NewStatus: "in_progress",
		}},
	}

	for _, e := range events {
		event, err := NewEvent(e.typ, e.payload)
		if err != nil {
			t.Fatalf("NewEvent(%s): %v", e.typ, err)
		}
		if err := es.Append(event); err != nil {
			t.Fatalf("Append(%s): %v", e.typ, err)
		}
	}

	// Replay
	var replayed []Event
	err = es.Replay(func(event Event) error {
		replayed = append(replayed, event)
		return nil
	})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	if len(replayed) != len(events) {
		t.Errorf("expected %d events, got %d", len(events), len(replayed))
	}

	// Verify order
	if replayed[0].Type != EventRequirementCreated {
		t.Errorf("first event should be %s, got %s", EventRequirementCreated, replayed[0].Type)
	}
	if replayed[1].Type != EventStoryCreated {
		t.Errorf("second event should be %s, got %s", EventStoryCreated, replayed[1].Type)
	}
}

func TestEventStore_Count(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	es, err := NewEventStore(path)
	if err != nil {
		t.Fatalf("NewEventStore: %v", err)
	}

	count, err := es.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 events, got %d", count)
	}

	event, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "t", Description: "d", Source: "s",
	})
	if err := es.Append(event); err != nil {
		t.Fatalf("Append: %v", err)
	}

	count, err = es.Count()
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 event, got %d", count)
	}
}

func TestEventStore_AppendRejectsInvalidPayload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	es, err := NewEventStore(path)
	if err != nil {
		t.Fatalf("NewEventStore: %v", err)
	}

	// Create event with unknown type
	event := Event{
		ID:      "test-id",
		Type:    EventType("bogus_type"),
		Payload: []byte(`{}`),
	}

	err = es.Append(event)
	if err == nil {
		t.Error("expected error for invalid event type")
	}
}

func TestEventStore_EmptyReplay(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	es, err := NewEventStore(path)
	if err != nil {
		t.Fatalf("NewEventStore: %v", err)
	}

	called := false
	err = es.Replay(func(_ Event) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}
	if called {
		t.Error("handler should not be called for empty store")
	}
}

func TestEventStore_Path(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")

	es, err := NewEventStore(path)
	if err != nil {
		t.Fatalf("NewEventStore: %v", err)
	}

	if es.Path() != path {
		t.Errorf("expected path %s, got %s", path, es.Path())
	}
}

func TestEventStore_CreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "events.jsonl")

	// Ensure subdir exists
	os.MkdirAll(filepath.Dir(path), 0755)

	es, err := NewEventStore(path)
	if err != nil {
		t.Fatalf("NewEventStore: %v", err)
	}

	if _, err := os.Stat(es.Path()); os.IsNotExist(err) {
		t.Error("expected event store file to be created")
	}
}
