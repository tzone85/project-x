package state

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewEvent_HasULID(t *testing.T) {
	evt := NewEvent(EventReqSubmitted, "agent-1", "", map[string]any{"id": "req-1"})
	if evt.ID == "" {
		t.Fatal("event ID should not be empty")
	}
	if evt.Type != EventReqSubmitted {
		t.Errorf("expected %s, got %s", EventReqSubmitted, evt.Type)
	}
}

func TestNewEvent_PayloadSerialization(t *testing.T) {
	payload := map[string]any{"id": "story-1", "complexity": 5}
	evt := NewEvent(EventStoryCreated, "planner", "story-1", payload)

	var decoded map[string]any
	if err := json.Unmarshal(evt.Payload, &decoded); err != nil {
		t.Fatalf("failed to decode payload: %v", err)
	}
	if decoded["id"] != "story-1" {
		t.Errorf("expected story-1, got %v", decoded["id"])
	}
}

func TestTypedPayload_StoryCreated(t *testing.T) {
	p := StoryCreatedPayload{
		ID:         "s-1",
		ReqID:      "r-1",
		Title:      "Add login",
		Complexity: 3,
		OwnedFiles: []string{"auth.go"},
		WaveHint:   "parallel",
		DependsOn:  []string{},
	}
	evt, err := NewTypedEvent(EventStoryCreated, "planner", "s-1", p)
	if err != nil {
		t.Fatalf("typed event error: %v", err)
	}

	var decoded StoryCreatedPayload
	if err := json.Unmarshal(evt.Payload, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Title != "Add login" {
		t.Errorf("expected 'Add login', got %q", decoded.Title)
	}
}

func TestEventTypes_UniqueStringValues(t *testing.T) {
	types := []EventType{
		EventReqSubmitted, EventReqAnalyzed, EventReqPlanned,
		EventReqPaused, EventReqResumed, EventReqCompleted,
		EventStoryCreated, EventStoryEstimated, EventStoryAssigned,
		EventStoryStarted, EventStoryProgress, EventStoryCompleted,
		EventStoryReviewRequested, EventStoryReviewPassed, EventStoryReviewFailed,
		EventStoryQAStarted, EventStoryQAPassed, EventStoryQAFailed,
		EventStoryPRCreated, EventStoryMerged,
		EventAgentSpawned, EventAgentStuck, EventAgentDied, EventAgentStale, EventAgentLost,
		EventEscalationCreated,
		EventBudgetWarning, EventBudgetExhausted,
	}

	seen := make(map[EventType]bool, len(types))
	for _, et := range types {
		if et == "" {
			t.Errorf("event type should not be empty string")
		}
		if seen[et] {
			t.Errorf("duplicate event type value: %s", et)
		}
		seen[et] = true
	}

	if len(seen) < 25 {
		t.Errorf("expected at least 25 unique event types, got %d", len(seen))
	}
}

func TestNewEvent_GeneratesUniqueIDs(t *testing.T) {
	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		evt := NewEvent(EventReqSubmitted, "agent-1", "", map[string]any{"i": i})
		if ids[evt.ID] {
			t.Fatalf("duplicate event ID generated: %s", evt.ID)
		}
		ids[evt.ID] = true
	}
}

func TestNewEvent_TimestampIsSet(t *testing.T) {
	before := time.Now().UTC()
	evt := NewEvent(EventReqSubmitted, "agent-1", "", map[string]any{})
	after := time.Now().UTC()

	if evt.Timestamp == "" {
		t.Fatal("timestamp should not be empty")
	}

	ts, err := time.Parse(time.RFC3339Nano, evt.Timestamp)
	if err != nil {
		t.Fatalf("failed to parse timestamp %q: %v", evt.Timestamp, err)
	}

	if ts.Before(before) || ts.After(after) {
		t.Errorf("timestamp %v not between %v and %v", ts, before, after)
	}
}

func TestNewEvent_SetsAgentAndStoryID(t *testing.T) {
	evt := NewEvent(EventStoryAssigned, "agent-42", "story-7", map[string]any{})
	if evt.AgentID != "agent-42" {
		t.Errorf("expected agent-42, got %q", evt.AgentID)
	}
	if evt.StoryID != "story-7" {
		t.Errorf("expected story-7, got %q", evt.StoryID)
	}
}

func TestNewEvent_NilPayload(t *testing.T) {
	evt := NewEvent(EventAgentSpawned, "agent-1", "", nil)
	if evt.Payload != nil {
		t.Errorf("expected nil payload for nil input, got %s", string(evt.Payload))
	}
}

func TestNewTypedEvent_ReqSubmittedPayload(t *testing.T) {
	p := ReqSubmittedPayload{
		ID:          "req-1",
		Title:       "Build auth module",
		Description: "Implement OAuth2 login flow",
		RepoPath:    "/home/user/myrepo",
	}
	evt, err := NewTypedEvent(EventReqSubmitted, "orchestrator", "", p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded ReqSubmittedPayload
	if err := json.Unmarshal(evt.Payload, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.ID != "req-1" {
		t.Errorf("expected req-1, got %q", decoded.ID)
	}
	if decoded.RepoPath != "/home/user/myrepo" {
		t.Errorf("expected /home/user/myrepo, got %q", decoded.RepoPath)
	}
}

func TestNewTypedEvent_StoryAssignedPayload(t *testing.T) {
	p := StoryAssignedPayload{
		AgentID: "agent-5",
		Wave:    2,
	}
	evt, err := NewTypedEvent(EventStoryAssigned, "supervisor", "story-3", p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded StoryAssignedPayload
	if err := json.Unmarshal(evt.Payload, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.AgentID != "agent-5" {
		t.Errorf("expected agent-5, got %q", decoded.AgentID)
	}
	if decoded.Wave != 2 {
		t.Errorf("expected wave 2, got %d", decoded.Wave)
	}
}

func TestNewTypedEvent_BudgetWarningPayload(t *testing.T) {
	p := BudgetWarningPayload{
		ReqID:      "req-1",
		StoryID:    "story-1",
		UsedUSD:    1.60,
		LimitUSD:   2.00,
		Percentage: 80,
	}
	evt, err := NewTypedEvent(EventBudgetWarning, "monitor", "story-1", p)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var decoded BudgetWarningPayload
	if err := json.Unmarshal(evt.Payload, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.UsedUSD != 1.60 {
		t.Errorf("expected 1.60, got %f", decoded.UsedUSD)
	}
	if decoded.Percentage != 80 {
		t.Errorf("expected 80, got %d", decoded.Percentage)
	}
}

func TestNewTypedEvent_UnmarshalablePayloadReturnsError(t *testing.T) {
	// channels cannot be marshaled to JSON
	_, err := NewTypedEvent(EventReqSubmitted, "agent", "", make(chan int))
	if err == nil {
		t.Fatal("expected error for unmarshalable payload")
	}
}

func TestEvent_JSONRoundTrip(t *testing.T) {
	original := NewEvent(EventStoryProgress, "agent-1", "story-5", map[string]any{
		"progress": 50,
		"message":  "halfway done",
	})

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var restored Event
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if restored.ID != original.ID {
		t.Errorf("ID mismatch: %q vs %q", restored.ID, original.ID)
	}
	if restored.Type != original.Type {
		t.Errorf("Type mismatch: %q vs %q", restored.Type, original.Type)
	}
	if restored.AgentID != original.AgentID {
		t.Errorf("AgentID mismatch: %q vs %q", restored.AgentID, original.AgentID)
	}
	if restored.StoryID != original.StoryID {
		t.Errorf("StoryID mismatch: %q vs %q", restored.StoryID, original.StoryID)
	}
	if restored.Timestamp != original.Timestamp {
		t.Errorf("Timestamp mismatch: %q vs %q", restored.Timestamp, original.Timestamp)
	}
}
