package state

import (
	"path/filepath"
	"testing"
)

func TestFileStore_AppendAndList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	fs, err := NewFileStore(path)
	if err != nil {
		t.Fatalf("new filestore: %v", err)
	}

	evt := NewEvent(EventReqSubmitted, "user", "", map[string]any{
		"id": "req-1", "title": "Add auth",
	})
	if err := fs.Append(evt); err != nil {
		t.Fatalf("append: %v", err)
	}

	events, err := fs.List(EventFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Type != EventReqSubmitted {
		t.Errorf("expected %s, got %s", EventReqSubmitted, events[0].Type)
	}
}

func TestFileStore_FilterByType(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	fs, _ := NewFileStore(path)

	fs.Append(NewEvent(EventReqSubmitted, "user", "", map[string]any{"id": "r1"}))
	fs.Append(NewEvent(EventStoryCreated, "planner", "s1", map[string]any{"id": "s1"}))
	fs.Append(NewEvent(EventReqSubmitted, "user", "", map[string]any{"id": "r2"}))

	events, _ := fs.List(EventFilter{Type: EventReqSubmitted})
	if len(events) != 2 {
		t.Fatalf("expected 2 req events, got %d", len(events))
	}
}

func TestFileStore_FilterByStoryID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	fs, _ := NewFileStore(path)

	fs.Append(NewEvent(EventStoryReviewFailed, "reviewer", "s1", map[string]any{}))
	fs.Append(NewEvent(EventStoryReviewFailed, "reviewer", "s2", map[string]any{}))
	fs.Append(NewEvent(EventStoryReviewFailed, "reviewer", "s1", map[string]any{}))

	events, _ := fs.List(EventFilter{StoryID: "s1"})
	if len(events) != 2 {
		t.Fatalf("expected 2 events for s1, got %d", len(events))
	}
}

func TestFileStore_FilterByAgentID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	fs, _ := NewFileStore(path)

	fs.Append(NewEvent(EventStoryReviewFailed, "reviewer", "s1", map[string]any{}))
	fs.Append(NewEvent(EventStoryReviewFailed, "monitor", "s1", map[string]any{}))

	events, _ := fs.List(EventFilter{AgentID: "reviewer", StoryID: "s1"})
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
}

func TestFileStore_Count(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	fs, _ := NewFileStore(path)

	fs.Append(NewEvent(EventStoryReviewFailed, "reviewer", "s1", map[string]any{}))
	fs.Append(NewEvent(EventStoryReviewFailed, "reviewer", "s1", map[string]any{}))
	fs.Append(NewEvent(EventStoryReviewFailed, "reviewer", "s2", map[string]any{}))

	count, _ := fs.Count(EventFilter{Type: EventStoryReviewFailed, StoryID: "s1"})
	if count != 2 {
		t.Fatalf("expected count 2, got %d", count)
	}
}

func TestFileStore_Limit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	fs, _ := NewFileStore(path)

	for i := 0; i < 10; i++ {
		fs.Append(NewEvent(EventStoryCreated, "planner", "", map[string]any{}))
	}

	events, _ := fs.List(EventFilter{Limit: 3})
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
}

func TestFileStore_PersistsToDisk(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	fs1, _ := NewFileStore(path)
	fs1.Append(NewEvent(EventReqSubmitted, "user", "", map[string]any{"id": "r1"}))

	// Re-open from same file
	fs2, _ := NewFileStore(path)
	events, _ := fs2.All()
	if len(events) != 1 {
		t.Fatalf("expected 1 event after reopen, got %d", len(events))
	}
}

func TestFileStore_All(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	fs, _ := NewFileStore(path)

	fs.Append(NewEvent(EventReqSubmitted, "user", "", map[string]any{"id": "r1"}))
	fs.Append(NewEvent(EventStoryCreated, "planner", "s1", map[string]any{"id": "s1"}))

	events, _ := fs.All()
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestFileStore_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	fs, _ := NewFileStore(path)

	events, _ := fs.All()
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestFileStore_CombinedFilters(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "events.jsonl")
	fs, _ := NewFileStore(path)

	fs.Append(NewEvent(EventStoryReviewFailed, "reviewer", "s1", map[string]any{}))
	fs.Append(NewEvent(EventStoryReviewFailed, "monitor", "s1", map[string]any{}))
	fs.Append(NewEvent(EventStoryCompleted, "reviewer", "s1", map[string]any{}))

	events, _ := fs.List(EventFilter{Type: EventStoryReviewFailed, AgentID: "reviewer", StoryID: "s1"})
	if len(events) != 1 {
		t.Fatalf("expected 1 event with all filters, got %d", len(events))
	}
}
