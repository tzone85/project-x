package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func newTestProjectionStore(t *testing.T) *ProjectionStore {
	t.Helper()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	ps, err := NewProjectionStore(dbPath)
	if err != nil {
		t.Fatalf("NewProjectionStore: %v", err)
	}
	t.Cleanup(func() { ps.Close() })
	return ps
}

func TestProjectionStore_WALMode(t *testing.T) {
	ps := newTestProjectionStore(t)

	var journalMode string
	err := ps.DB().QueryRow("PRAGMA journal_mode").Scan(&journalMode)
	if err != nil {
		t.Fatalf("PRAGMA journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("expected WAL mode, got %s", journalMode)
	}
}

func TestProjectionStore_ApplyStoryCreated(t *testing.T) {
	ps := newTestProjectionStore(t)

	// First create a requirement
	reqEvent, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "Req 1", Description: "Test", Source: "test",
	})
	if err := ps.ApplyEvent(reqEvent); err != nil {
		t.Fatalf("apply requirement: %v", err)
	}

	// Create a story
	storyEvent, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
		StoryID: "story-1", RequirementID: "req-1", Title: "Story 1",
		Description: "A story", AcceptanceCriteria: "It works",
		Complexity: 5, OwnedFiles: []string{"a.go", "b.go"}, DependsOn: []string{},
	})
	if err := ps.ApplyEvent(storyEvent); err != nil {
		t.Fatalf("apply story: %v", err)
	}

	// Query it back
	story, err := ps.GetStory("story-1")
	if err != nil {
		t.Fatalf("GetStory: %v", err)
	}
	if story == nil {
		t.Fatal("story not found")
	}
	if story.Title != "Story 1" {
		t.Errorf("expected title 'Story 1', got %s", story.Title)
	}
	if story.Complexity != 5 {
		t.Errorf("expected complexity 5, got %d", story.Complexity)
	}
	if len(story.OwnedFiles) != 2 {
		t.Errorf("expected 2 owned files, got %d", len(story.OwnedFiles))
	}
	if story.Status != "planned" {
		t.Errorf("expected status 'planned', got %s", story.Status)
	}
}

func TestProjectionStore_ApplyStoryStatusChanged(t *testing.T) {
	ps := newTestProjectionStore(t)

	// Setup
	reqEvent, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "Req", Description: "", Source: "test",
	})
	ps.ApplyEvent(reqEvent)

	storyEvent, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
		StoryID: "story-1", RequirementID: "req-1", Title: "S1",
		Complexity: 3, OwnedFiles: []string{}, DependsOn: []string{},
	})
	ps.ApplyEvent(storyEvent)

	// Change status
	statusEvent, _ := NewEvent(EventStoryStatusChanged, StoryStatusChangedPayload{
		StoryID: "story-1", OldStatus: "planned", NewStatus: "in_progress",
	})
	if err := ps.ApplyEvent(statusEvent); err != nil {
		t.Fatalf("apply status change: %v", err)
	}

	story, _ := ps.GetStory("story-1")
	if story.Status != "in_progress" {
		t.Errorf("expected status 'in_progress', got %s", story.Status)
	}
}

func TestProjectionStore_ApplyAgentAssigned(t *testing.T) {
	ps := newTestProjectionStore(t)

	// Setup requirement + story
	reqEvent, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "Req", Description: "", Source: "test",
	})
	ps.ApplyEvent(reqEvent)

	storyEvent, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
		StoryID: "story-1", RequirementID: "req-1", Title: "S1",
		Complexity: 3, OwnedFiles: []string{}, DependsOn: []string{},
	})
	ps.ApplyEvent(storyEvent)

	// Assign agent
	agentEvent, _ := NewEvent(EventAgentAssigned, AgentAssignedPayload{
		AgentID: "agent-1", StoryID: "story-1", Role: "senior",
		Runtime: "claude-code", Session: "session-1",
	})
	if err := ps.ApplyEvent(agentEvent); err != nil {
		t.Fatalf("apply agent assigned: %v", err)
	}

	agent, err := ps.GetAgent("agent-1")
	if err != nil {
		t.Fatalf("GetAgent: %v", err)
	}
	if agent == nil {
		t.Fatal("agent not found")
	}
	if agent.Role != "senior" {
		t.Errorf("expected role 'senior', got %s", agent.Role)
	}
	if agent.Status != "working" {
		t.Errorf("expected status 'working', got %s", agent.Status)
	}

	// Verify story's agent_id updated
	story, _ := ps.GetStory("story-1")
	if story.AgentID != "agent-1" {
		t.Errorf("expected story agent_id 'agent-1', got %s", story.AgentID)
	}
}

func TestProjectionStore_ApplyTokenUsage(t *testing.T) {
	ps := newTestProjectionStore(t)

	event, _ := NewEvent(EventTokenUsageRecorded, TokenUsageRecordedPayload{
		StoryID: "story-1", RequirementID: "req-1", AgentID: "agent-1",
		Model: "claude-sonnet", InputTokens: 1000, OutputTokens: 500,
		CostUSD: 0.105, Stage: "review",
	})
	if err := ps.ApplyEvent(event); err != nil {
		t.Fatalf("apply token usage: %v", err)
	}

	total, err := ps.GetStoryTotalCost("story-1")
	if err != nil {
		t.Fatalf("GetStoryTotalCost: %v", err)
	}
	if total != 0.105 {
		t.Errorf("expected cost 0.105, got %f", total)
	}

	reqTotal, err := ps.GetRequirementTotalCost("req-1")
	if err != nil {
		t.Fatalf("GetRequirementTotalCost: %v", err)
	}
	if reqTotal != 0.105 {
		t.Errorf("expected req cost 0.105, got %f", reqTotal)
	}
}

func TestProjectionStore_ApplyEscalation(t *testing.T) {
	ps := newTestProjectionStore(t)

	// Setup
	reqEvent, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "Req", Description: "", Source: "test",
	})
	ps.ApplyEvent(reqEvent)
	storyEvent, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
		StoryID: "story-1", RequirementID: "req-1", Title: "S1",
		Complexity: 3, OwnedFiles: []string{}, DependsOn: []string{},
	})
	ps.ApplyEvent(storyEvent)

	escEvent, _ := NewEvent(EventEscalationCreated, EscalationCreatedPayload{
		EscalationID: "esc-1", StoryID: "story-1", Reason: "QA failed 3 times",
		FromRole: "junior", ToRole: "senior",
	})
	if err := ps.ApplyEvent(escEvent); err != nil {
		t.Fatalf("apply escalation: %v", err)
	}

	escalations, err := ps.ListEscalations(DefaultPageParams())
	if err != nil {
		t.Fatalf("ListEscalations: %v", err)
	}
	if len(escalations) != 1 {
		t.Fatalf("expected 1 escalation, got %d", len(escalations))
	}
	if escalations[0].Reason != "QA failed 3 times" {
		t.Errorf("expected reason 'QA failed 3 times', got %s", escalations[0].Reason)
	}
}

func TestProjectionStore_ApplyPipelineRun(t *testing.T) {
	ps := newTestProjectionStore(t)

	// Setup
	reqEvent, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "Req", Description: "", Source: "test",
	})
	ps.ApplyEvent(reqEvent)
	storyEvent, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
		StoryID: "story-1", RequirementID: "req-1", Title: "S1",
		Complexity: 3, OwnedFiles: []string{}, DependsOn: []string{},
	})
	ps.ApplyEvent(storyEvent)

	startEvent, _ := NewEvent(EventPipelineRunStarted, PipelineRunStartedPayload{
		StoryID: "story-1", Stage: "review", Attempt: 1,
	})
	if err := ps.ApplyEvent(startEvent); err != nil {
		t.Fatalf("apply pipeline start: %v", err)
	}

	updateEvent, _ := NewEvent(EventPipelineRunUpdated, PipelineRunUpdatedPayload{
		StoryID: "story-1", Stage: "review", Status: "passed",
		Attempt: 1, DurationMs: 5000,
	})
	if err := ps.ApplyEvent(updateEvent); err != nil {
		t.Fatalf("apply pipeline update: %v", err)
	}

	runs, err := ps.ListPipelineRuns(DefaultPageParams())
	if err != nil {
		t.Fatalf("ListPipelineRuns: %v", err)
	}
	if len(runs) != 1 {
		t.Fatalf("expected 1 run, got %d", len(runs))
	}
	if runs[0].Status != "passed" {
		t.Errorf("expected status 'passed', got %s", runs[0].Status)
	}
}

func TestProjectionStore_ApplySessionHealth(t *testing.T) {
	ps := newTestProjectionStore(t)

	event, _ := NewEvent(EventSessionHealthChanged, SessionHealthChangedPayload{
		SessionName: "agent-session-1", OldStatus: "unknown", NewStatus: "healthy",
		PanePID: 12345, RecoveryAttempt: 0,
	})
	if err := ps.ApplyEvent(event); err != nil {
		t.Fatalf("apply session health: %v", err)
	}

	// Update to stale
	event2, _ := NewEvent(EventSessionHealthChanged, SessionHealthChangedPayload{
		SessionName: "agent-session-1", OldStatus: "healthy", NewStatus: "stale",
		PanePID: 12345, RecoveryAttempt: 1,
	})
	if err := ps.ApplyEvent(event2); err != nil {
		t.Fatalf("apply session health update: %v", err)
	}
}

func TestProjectionStore_ListRequirementsPagination(t *testing.T) {
	ps := newTestProjectionStore(t)

	// Create 5 requirements
	for i := 0; i < 5; i++ {
		event, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
			RequirementID: "req-" + string(rune('a'+i)),
			Title:         "Req " + string(rune('A'+i)),
			Description:   "", Source: "test",
		})
		ps.ApplyEvent(event)
	}

	// Page 1: limit 2
	page1, err := ps.ListRequirements(PageParams{Limit: 2, Offset: 0})
	if err != nil {
		t.Fatalf("ListRequirements page 1: %v", err)
	}
	if len(page1) != 2 {
		t.Errorf("expected 2 items, got %d", len(page1))
	}

	// Page 2: limit 2, offset 2
	page2, err := ps.ListRequirements(PageParams{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("ListRequirements page 2: %v", err)
	}
	if len(page2) != 2 {
		t.Errorf("expected 2 items, got %d", len(page2))
	}

	// Page 3: limit 2, offset 4
	page3, err := ps.ListRequirements(PageParams{Limit: 2, Offset: 4})
	if err != nil {
		t.Fatalf("ListRequirements page 3: %v", err)
	}
	if len(page3) != 1 {
		t.Errorf("expected 1 item, got %d", len(page3))
	}
}

func TestProjectionStore_ListStoriesByRequirement(t *testing.T) {
	ps := newTestProjectionStore(t)

	// Create 2 requirements
	for _, rid := range []string{"req-1", "req-2"} {
		event, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
			RequirementID: rid, Title: rid, Description: "", Source: "test",
		})
		ps.ApplyEvent(event)
	}

	// Create stories in each
	for i := 0; i < 3; i++ {
		event, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
			StoryID: "story-1-" + string(rune('a'+i)), RequirementID: "req-1",
			Title: "S", Complexity: 3, OwnedFiles: []string{}, DependsOn: []string{},
		})
		ps.ApplyEvent(event)
	}
	event, _ := NewEvent(EventStoryCreated, StoryCreatedPayload{
		StoryID: "story-2-a", RequirementID: "req-2",
		Title: "S", Complexity: 3, OwnedFiles: []string{}, DependsOn: []string{},
	})
	ps.ApplyEvent(event)

	// Query req-1 stories
	stories, err := ps.ListStoriesByRequirement("req-1", DefaultPageParams())
	if err != nil {
		t.Fatalf("ListStoriesByRequirement: %v", err)
	}
	if len(stories) != 3 {
		t.Errorf("expected 3 stories for req-1, got %d", len(stories))
	}

	// Query req-2 stories
	stories2, err := ps.ListStoriesByRequirement("req-2", DefaultPageParams())
	if err != nil {
		t.Fatalf("ListStoriesByRequirement: %v", err)
	}
	if len(stories2) != 1 {
		t.Errorf("expected 1 story for req-2, got %d", len(stories2))
	}
}

func TestProjectionStore_GetDailyTotalCost(t *testing.T) {
	ps := newTestProjectionStore(t)

	event, _ := NewEvent(EventTokenUsageRecorded, TokenUsageRecordedPayload{
		StoryID: "s1", RequirementID: "r1", AgentID: "a1",
		Model: "claude-sonnet", InputTokens: 1000, OutputTokens: 500,
		CostUSD: 1.50, Stage: "review",
	})
	ps.ApplyEvent(event)

	today := time.Now().UTC()
	total, err := ps.GetDailyTotalCost(today)
	if err != nil {
		t.Fatalf("GetDailyTotalCost: %v", err)
	}
	if total != 1.50 {
		t.Errorf("expected 1.50, got %f", total)
	}

	// Different day should be 0
	yesterday := today.AddDate(0, 0, -1)
	total2, err := ps.GetDailyTotalCost(yesterday)
	if err != nil {
		t.Fatalf("GetDailyTotalCost yesterday: %v", err)
	}
	if total2 != 0 {
		t.Errorf("expected 0 for yesterday, got %f", total2)
	}
}

func TestProjectionStore_ReadOnlyConnection(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	ps, err := NewProjectionStore(dbPath)
	if err != nil {
		t.Fatalf("NewProjectionStore: %v", err)
	}
	defer ps.Close()

	// Write some data
	event, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "Req 1", Description: "Test", Source: "test",
	})
	ps.ApplyEvent(event)

	// Open read-only connection
	roDB, err := NewReadOnlyConnection(dbPath)
	if err != nil {
		t.Fatalf("NewReadOnlyConnection: %v", err)
	}
	defer roDB.Close()

	ro := NewReadOnlyQueries(roDB)

	req, err := ro.GetRequirement("req-1")
	if err != nil {
		t.Fatalf("GetRequirement (read-only): %v", err)
	}
	if req == nil {
		t.Fatal("requirement not found via read-only connection")
	}
	if req.Title != "Req 1" {
		t.Errorf("expected title 'Req 1', got %s", req.Title)
	}
}

func TestProjectionStore_GetStoryNotFound(t *testing.T) {
	ps := newTestProjectionStore(t)

	story, err := ps.GetStory("nonexistent")
	if err != nil {
		t.Fatalf("GetStory: %v", err)
	}
	if story != nil {
		t.Error("expected nil for nonexistent story")
	}
}

func TestProjectionStore_ListEventsStored(t *testing.T) {
	ps := newTestProjectionStore(t)

	event, _ := NewEvent(EventRequirementCreated, RequirementCreatedPayload{
		RequirementID: "req-1", Title: "Req 1", Description: "", Source: "test",
	})
	ps.ApplyEvent(event)

	events, err := ps.ListEvents(DefaultPageParams())
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestNewProjectionStore_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "new.db")

	ps, err := NewProjectionStore(dbPath)
	if err != nil {
		t.Fatalf("NewProjectionStore: %v", err)
	}
	ps.Close()

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		t.Error("expected db file to be created")
	}
}
