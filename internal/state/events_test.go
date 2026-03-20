package state

import (
	"encoding/json"
	"testing"
)

func TestNewEvent(t *testing.T) {
	payload := StoryCreatedPayload{
		StoryID:       "story-1",
		RequirementID: "req-1",
		Title:         "Test Story",
		Description:   "A test story",
		Complexity:    3,
		OwnedFiles:    []string{"file.go"},
		DependsOn:     []string{},
	}

	event, err := NewEvent(EventStoryCreated, payload)
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}

	if event.ID == "" {
		t.Error("event ID should not be empty")
	}
	if event.Type != EventStoryCreated {
		t.Errorf("expected type %s, got %s", EventStoryCreated, event.Type)
	}
	if event.Timestamp.IsZero() {
		t.Error("timestamp should not be zero")
	}
	if len(event.Payload) == 0 {
		t.Error("payload should not be empty")
	}
}

func TestDecodePayload(t *testing.T) {
	original := StoryCreatedPayload{
		StoryID:       "story-1",
		RequirementID: "req-1",
		Title:         "Test Story",
		Complexity:    5,
		OwnedFiles:    []string{"a.go", "b.go"},
		DependsOn:     []string{"story-0"},
	}

	event, err := NewEvent(EventStoryCreated, original)
	if err != nil {
		t.Fatalf("NewEvent failed: %v", err)
	}

	decoded, err := DecodePayload[StoryCreatedPayload](event)
	if err != nil {
		t.Fatalf("DecodePayload failed: %v", err)
	}

	if decoded.StoryID != original.StoryID {
		t.Errorf("StoryID: expected %s, got %s", original.StoryID, decoded.StoryID)
	}
	if decoded.Title != original.Title {
		t.Errorf("Title: expected %s, got %s", original.Title, decoded.Title)
	}
	if decoded.Complexity != original.Complexity {
		t.Errorf("Complexity: expected %d, got %d", original.Complexity, decoded.Complexity)
	}
	if len(decoded.OwnedFiles) != len(original.OwnedFiles) {
		t.Errorf("OwnedFiles length: expected %d, got %d", len(original.OwnedFiles), len(decoded.OwnedFiles))
	}
}

func TestDecodePayload_TypeMismatch(t *testing.T) {
	// Create an event with a payload that doesn't match the decode target
	event := Event{
		ID:      "test-id",
		Type:    EventStoryCreated,
		Payload: json.RawMessage(`{"not_a_valid_field": true}`),
	}

	// This should succeed because Go's json.Unmarshal ignores unknown fields
	// and zero-values missing fields. The real validation is in ValidatePayload.
	_, err := DecodePayload[StoryCreatedPayload](event)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestDecodePayload_InvalidJSON(t *testing.T) {
	event := Event{
		ID:      "test-id",
		Type:    EventStoryCreated,
		Payload: json.RawMessage(`{invalid json`),
	}

	_, err := DecodePayload[StoryCreatedPayload](event)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestValidatePayload(t *testing.T) {
	// Valid payload
	event, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
		StoryID: "s1", RequirementID: "r1", Title: "t",
	})
	if err := ValidatePayload(event); err != nil {
		t.Errorf("valid payload should not error: %v", err)
	}
}

func TestValidatePayload_UnknownType(t *testing.T) {
	event := Event{
		ID:      "test-id",
		Type:    EventType("unknown_type"),
		Payload: json.RawMessage(`{}`),
	}

	err := ValidatePayload(event)
	if err == nil {
		t.Error("expected error for unknown event type")
	}
}

func TestValidatePayload_InvalidPayload(t *testing.T) {
	event := Event{
		ID:      "test-id",
		Type:    EventStoryCreated,
		Payload: json.RawMessage(`{bad json`),
	}

	err := ValidatePayload(event)
	if err == nil {
		t.Error("expected error for invalid JSON payload")
	}
}

func TestAllEventTypesHaveRegisteredPayloads(t *testing.T) {
	types := []EventType{
		EventStoryCreated, EventStoryUpdated, EventStoryStatusChanged,
		EventAgentAssigned, EventAgentStatusChanged, EventAgentDied,
		EventAgentStale, EventAgentLost,
		EventRequirementCreated, EventRequirementUpdated,
		EventEscalationCreated,
		EventPipelineRunStarted, EventPipelineRunUpdated,
		EventBudgetWarning, EventTokenUsageRecorded,
		EventSessionHealthChanged,
	}

	for _, et := range types {
		if _, ok := payloadRegistry[et]; !ok {
			t.Errorf("event type %s not registered in payloadRegistry", et)
		}
	}
}
