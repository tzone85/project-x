package state

import (
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func TestProjectionStore_ApplyAgentStatusChanged(t *testing.T) {
	ps := newTestProjectionStore(t)

	// Setup agent via assignment
	reqEvent, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "R", Description: "", Source: "test",
	})
	ps.ApplyEvent(reqEvent)
	storyEvent, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
		StoryID: "story-1", RequirementID: "req-1", Title: "S",
		Complexity: 3, OwnedFiles: []string{}, DependsOn: []string{},
	})
	ps.ApplyEvent(storyEvent)
	agentEvent, _ := NewEvent(EventAgentAssigned, AgentAssignedPayload{
		AgentID: "agent-1", StoryID: "story-1", Role: "senior",
		Runtime: "claude-code", Session: "sess-1",
	})
	ps.ApplyEvent(agentEvent)

	// Now change status
	statusEvent, _ := NewEvent(EventAgentStatusChanged, AgentStatusChangedPayload{
		AgentID: "agent-1", OldStatus: "working", NewStatus: "idle",
	})
	if err := ps.ApplyEvent(statusEvent); err != nil {
		t.Fatalf("apply agent status changed: %v", err)
	}

	agent, _ := ps.GetAgent("agent-1")
	if agent.Status != "idle" {
		t.Errorf("expected status 'idle', got %s", agent.Status)
	}
}

func TestProjectionStore_ListStories(t *testing.T) {
	ps := newTestProjectionStore(t)

	reqEvent, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "R", Description: "", Source: "test",
	})
	ps.ApplyEvent(reqEvent)

	for i := 0; i < 3; i++ {
		event, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
			StoryID: "story-" + string(rune('a'+i)), RequirementID: "req-1",
			Title: "S", Complexity: 2, OwnedFiles: []string{}, DependsOn: []string{},
		})
		ps.ApplyEvent(event)
	}

	stories, err := ps.ListStories(DefaultPageParams())
	if err != nil {
		t.Fatalf("ListStories: %v", err)
	}
	if len(stories) != 3 {
		t.Errorf("expected 3 stories, got %d", len(stories))
	}
}

func TestProjectionStore_ListAgents(t *testing.T) {
	ps := newTestProjectionStore(t)

	reqEvent, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "R", Description: "", Source: "test",
	})
	ps.ApplyEvent(reqEvent)
	storyEvent, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
		StoryID: "story-1", RequirementID: "req-1", Title: "S",
		Complexity: 3, OwnedFiles: []string{}, DependsOn: []string{},
	})
	ps.ApplyEvent(storyEvent)

	agentEvent, _ := NewEvent(EventAgentAssigned, AgentAssignedPayload{
		AgentID: "agent-1", StoryID: "story-1", Role: "senior",
		Runtime: "claude-code", Session: "sess-1",
	})
	ps.ApplyEvent(agentEvent)

	agents, err := ps.ListAgents(DefaultPageParams())
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(agents) != 1 {
		t.Errorf("expected 1 agent, got %d", len(agents))
	}
}

func TestProjectionStore_ListTokenUsage(t *testing.T) {
	ps := newTestProjectionStore(t)

	event, _ := NewEvent(EventTokenUsageRecorded, TokenUsageRecordedPayload{
		StoryID: "s1", RequirementID: "r1", AgentID: "a1",
		Model: "claude-sonnet", InputTokens: 100, OutputTokens: 50,
		CostUSD: 0.01, Stage: "review",
	})
	ps.ApplyEvent(event)

	usage, err := ps.ListTokenUsage(DefaultPageParams())
	if err != nil {
		t.Fatalf("ListTokenUsage: %v", err)
	}
	if len(usage) != 1 {
		t.Errorf("expected 1 usage, got %d", len(usage))
	}
}

func TestReadOnlyQueries_AllMethods(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	ps, err := NewProjectionStore(dbPath)
	if err != nil {
		t.Fatalf("NewProjectionStore: %v", err)
	}

	// Create test data
	reqEvent, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "Req", Description: "", Source: "test",
	})
	ps.ApplyEvent(reqEvent)

	storyEvent, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
		StoryID: "story-1", RequirementID: "req-1", Title: "S",
		Complexity: 3, OwnedFiles: []string{}, DependsOn: []string{},
	})
	ps.ApplyEvent(storyEvent)

	agentEvent, _ := NewEvent(EventAgentAssigned, AgentAssignedPayload{
		AgentID: "agent-1", StoryID: "story-1", Role: "senior",
		Runtime: "claude-code", Session: "sess-1",
	})
	ps.ApplyEvent(agentEvent)

	escEvent, _ := NewEvent(EventEscalationCreated, EscalationCreatedPayload{
		EscalationID: "esc-1", StoryID: "story-1", Reason: "test",
		FromRole: "junior", ToRole: "senior",
	})
	ps.ApplyEvent(escEvent)

	tokenEvent, _ := NewEvent(EventTokenUsageRecorded, TokenUsageRecordedPayload{
		StoryID: "story-1", RequirementID: "req-1", AgentID: "agent-1",
		Model: "claude-sonnet", InputTokens: 100, OutputTokens: 50,
		CostUSD: 0.01, Stage: "review",
	})
	ps.ApplyEvent(tokenEvent)

	pipeEvent, _ := NewEvent(EventPipelineRunStarted, PipelineRunStartedPayload{
		StoryID: "story-1", Stage: "review", Attempt: 1,
	})
	ps.ApplyEvent(pipeEvent)

	// Open read-only connection
	roDB, err := NewReadOnlyConnection(dbPath)
	if err != nil {
		t.Fatalf("NewReadOnlyConnection: %v", err)
	}
	ro := NewReadOnlyQueries(roDB)
	defer ro.Close()

	page := DefaultPageParams()

	// Test all ReadOnly methods
	if _, err := ro.ListRequirements(page); err != nil {
		t.Errorf("ro.ListRequirements: %v", err)
	}
	if _, err := ro.GetRequirement("req-1"); err != nil {
		t.Errorf("ro.GetRequirement: %v", err)
	}
	if _, err := ro.ListStories(page); err != nil {
		t.Errorf("ro.ListStories: %v", err)
	}
	if _, err := ro.GetStory("story-1"); err != nil {
		t.Errorf("ro.GetStory: %v", err)
	}
	if _, err := ro.ListStoriesByRequirement("req-1", page); err != nil {
		t.Errorf("ro.ListStoriesByRequirement: %v", err)
	}
	if _, err := ro.ListAgents(page); err != nil {
		t.Errorf("ro.ListAgents: %v", err)
	}
	if _, err := ro.GetAgent("agent-1"); err != nil {
		t.Errorf("ro.GetAgent: %v", err)
	}
	if _, err := ro.ListEscalations(page); err != nil {
		t.Errorf("ro.ListEscalations: %v", err)
	}
	if _, err := ro.ListTokenUsage(page); err != nil {
		t.Errorf("ro.ListTokenUsage: %v", err)
	}
	if _, err := ro.GetStoryTotalCost("story-1"); err != nil {
		t.Errorf("ro.GetStoryTotalCost: %v", err)
	}
	if _, err := ro.GetRequirementTotalCost("req-1"); err != nil {
		t.Errorf("ro.GetRequirementTotalCost: %v", err)
	}
	if _, err := ro.GetDailyTotalCost(time.Now()); err != nil {
		t.Errorf("ro.GetDailyTotalCost: %v", err)
	}
	if _, err := ro.ListPipelineRuns(page); err != nil {
		t.Errorf("ro.ListPipelineRuns: %v", err)
	}
	if _, err := ro.ListEvents(page); err != nil {
		t.Errorf("ro.ListEvents: %v", err)
	}

	ps.Close()
}

func TestProjector_WithLogger(t *testing.T) {
	ps := newTestProjectionStore(t)
	logger := slog.Default()

	projector := NewProjector(ps, WithLogger(logger))
	if projector.logger != logger {
		t.Error("expected custom logger to be set")
	}
}

func TestProjectionStore_UnknownEventTypeGoesToEventsTable(t *testing.T) {
	ps := newTestProjectionStore(t)

	// Create a valid but unhandled event type using BudgetWarning
	// (not in the ApplyEvent switch but still in payloadRegistry)
	event, _ := NewEvent(EventBudgetWarning, BudgetWarningPayload{
		StoryID: "s1", CurrentCost: 1.5, BudgetLimit: 2.0, Percentage: 75,
	})
	if err := ps.ApplyEvent(event); err != nil {
		t.Fatalf("apply unknown type: %v", err)
	}

	events, err := ps.ListEvents(DefaultPageParams())
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event in events table, got %d", len(events))
	}
}
