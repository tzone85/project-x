package state

import (
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestFullEventReplayRebuild(t *testing.T) {
	dir := t.TempDir()
	jsonlPath := filepath.Join(dir, "events.jsonl")
	dbPath := filepath.Join(dir, "test.db")

	// Phase 1: Write events to JSONL store and project them
	es, err := NewEventStore(jsonlPath)
	if err != nil {
		t.Fatalf("NewEventStore: %v", err)
	}

	ps, err := NewProjectionStore(dbPath)
	if err != nil {
		t.Fatalf("NewProjectionStore: %v", err)
	}

	// Create a sequence of events
	events := []struct {
		typ     EventType
		payload any
	}{
		{EventRequirementCreated, RequirementCreatedPayload{
			RequirementID: "req-1", Title: "Auth System", Description: "Build auth", Source: "spec.md",
		}},
		{EventStoryCreated, StoryCreatedPayload{
			StoryID: "story-1", RequirementID: "req-1", Title: "Login endpoint",
			Description: "Create POST /login", AcceptanceCriteria: "Returns JWT",
			Complexity: 5, OwnedFiles: []string{"auth/login.go"}, DependsOn: []string{},
		}},
		{EventStoryCreated, StoryCreatedPayload{
			StoryID: "story-2", RequirementID: "req-1", Title: "Token refresh",
			Description: "Create POST /refresh", AcceptanceCriteria: "Extends JWT",
			Complexity: 3, OwnedFiles: []string{"auth/refresh.go"}, DependsOn: []string{"story-1"},
		}},
		{EventAgentAssigned, AgentAssignedPayload{
			AgentID: "agent-1", StoryID: "story-1", Role: "senior",
			Runtime: "claude-code", Session: "tmux-session-1",
		}},
		{EventStoryStatusChanged, StoryStatusChangedPayload{
			StoryID: "story-1", OldStatus: "planned", NewStatus: "in_progress",
		}},
		{EventTokenUsageRecorded, TokenUsageRecordedPayload{
			StoryID: "story-1", RequirementID: "req-1", AgentID: "agent-1",
			Model: "claude-sonnet", InputTokens: 2000, OutputTokens: 800,
			CostUSD: 0.42, Stage: "coding",
		}},
		{EventPipelineRunStarted, PipelineRunStartedPayload{
			StoryID: "story-1", Stage: "review", Attempt: 1,
		}},
		{EventPipelineRunUpdated, PipelineRunUpdatedPayload{
			StoryID: "story-1", Stage: "review", Status: "passed",
			Attempt: 1, DurationMs: 3000,
		}},
		{EventStoryStatusChanged, StoryStatusChangedPayload{
			StoryID: "story-1", OldStatus: "in_progress", NewStatus: "merged",
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
		if err := ps.ApplyEvent(event); err != nil {
			t.Fatalf("ApplyEvent(%s): %v", e.typ, err)
		}
	}

	// Verify original projection
	story1, _ := ps.GetStory("story-1")
	if story1.Status != "merged" {
		t.Errorf("original: expected status 'merged', got %s", story1.Status)
	}
	if story1.AgentID != "agent-1" {
		t.Errorf("original: expected agent 'agent-1', got %s", story1.AgentID)
	}

	cost, _ := ps.GetStoryTotalCost("story-1")
	if cost != 0.42 {
		t.Errorf("original: expected cost 0.42, got %f", cost)
	}
	ps.Close()

	// Phase 2: Rebuild projection from scratch using JSONL replay
	dbPath2 := filepath.Join(dir, "rebuilt.db")
	ps2, err := NewProjectionStore(dbPath2)
	if err != nil {
		t.Fatalf("NewProjectionStore (rebuild): %v", err)
	}
	defer ps2.Close()

	err = es.Replay(func(event Event) error {
		return ps2.ApplyEvent(event)
	})
	if err != nil {
		t.Fatalf("Replay: %v", err)
	}

	// Verify rebuilt projection matches original
	story1r, _ := ps2.GetStory("story-1")
	if story1r == nil {
		t.Fatal("rebuilt: story-1 not found")
	}
	if story1r.Status != "merged" {
		t.Errorf("rebuilt: expected status 'merged', got %s", story1r.Status)
	}
	if story1r.AgentID != "agent-1" {
		t.Errorf("rebuilt: expected agent 'agent-1', got %s", story1r.AgentID)
	}

	story2r, _ := ps2.GetStory("story-2")
	if story2r == nil {
		t.Fatal("rebuilt: story-2 not found")
	}
	if story2r.Title != "Token refresh" {
		t.Errorf("rebuilt: expected title 'Token refresh', got %s", story2r.Title)
	}
	if len(story2r.DependsOn) != 1 || story2r.DependsOn[0] != "story-1" {
		t.Errorf("rebuilt: expected depends_on ['story-1'], got %v", story2r.DependsOn)
	}

	agent, _ := ps2.GetAgent("agent-1")
	if agent == nil {
		t.Fatal("rebuilt: agent not found")
	}
	if agent.Role != "senior" {
		t.Errorf("rebuilt: expected role 'senior', got %s", agent.Role)
	}

	cost2, _ := ps2.GetStoryTotalCost("story-1")
	if cost2 != 0.42 {
		t.Errorf("rebuilt: expected cost 0.42, got %f", cost2)
	}

	req, _ := ps2.GetRequirement("req-1")
	if req == nil {
		t.Fatal("rebuilt: requirement not found")
	}
	if req.Title != "Auth System" {
		t.Errorf("rebuilt: expected title 'Auth System', got %s", req.Title)
	}

	runs, _ := ps2.ListPipelineRuns(DefaultPageParams())
	if len(runs) != 1 {
		t.Errorf("rebuilt: expected 1 pipeline run, got %d", len(runs))
	}
	if len(runs) > 0 && runs[0].Status != "passed" {
		t.Errorf("rebuilt: expected run status 'passed', got %s", runs[0].Status)
	}
}

func TestPagination_Normalize(t *testing.T) {
	tests := []struct {
		name     string
		input    PageParams
		expected PageParams
	}{
		{"zero limit gets default", PageParams{0, 0}, PageParams{50, 0}},
		{"negative limit gets default", PageParams{-1, 0}, PageParams{50, 0}},
		{"over max gets capped", PageParams{2000, 0}, PageParams{1000, 0}},
		{"negative offset gets zero", PageParams{10, -5}, PageParams{10, 0}},
		{"valid params unchanged", PageParams{25, 10}, PageParams{25, 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.input.Normalize()
			if got != tt.expected {
				t.Errorf("Normalize(%+v) = %+v, want %+v", tt.input, got, tt.expected)
			}
		})
	}
}
