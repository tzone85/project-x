package state

import (
	"encoding/json"
	"testing"
)

// newTestStore creates an in-memory SQLiteStore for testing.
func newTestStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

// mustMarshal marshals v to JSON or fails the test.
func mustMarshal(t *testing.T, v any) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return data
}

func TestSQLiteStore_ProjectReqSubmitted(t *testing.T) {
	store := newTestStore(t)

	evt := Event{
		ID:        "evt-1",
		Type:      EventReqSubmitted,
		Timestamp: "2026-01-01T00:00:00Z",
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID:          "req-1",
			Title:       "Build a widget",
			Description: "We need a widget that does things",
			RepoPath:    "/repos/widget",
		}),
	}

	if err := store.Project(evt); err != nil {
		t.Fatalf("Project: %v", err)
	}

	req, err := store.GetRequirement("req-1")
	if err != nil {
		t.Fatalf("GetRequirement: %v", err)
	}

	if req.ID != "req-1" {
		t.Errorf("ID: got %q, want %q", req.ID, "req-1")
	}
	if req.Title != "Build a widget" {
		t.Errorf("Title: got %q, want %q", req.Title, "Build a widget")
	}
	if req.Description != "We need a widget that does things" {
		t.Errorf("Description: got %q, want %q", req.Description, "We need a widget that does things")
	}
	if req.Status != "pending" {
		t.Errorf("Status: got %q, want %q", req.Status, "pending")
	}
	if req.RepoPath != "/repos/widget" {
		t.Errorf("RepoPath: got %q, want %q", req.RepoPath, "/repos/widget")
	}
}

func TestSQLiteStore_ProjectStoryLifecycle(t *testing.T) {
	store := newTestStore(t)

	// Submit requirement first
	store.Project(Event{
		ID:   "evt-0",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID:          "req-1",
			Title:       "Req",
			Description: "Desc",
			RepoPath:    "/repo",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})

	// Create story
	store.Project(Event{
		ID:      "evt-1",
		Type:    EventStoryCreated,
		StoryID: "story-1",
		Payload: mustMarshal(t, StoryCreatedPayload{
			ID:                 "story-1",
			ReqID:              "req-1",
			Title:              "Implement widget",
			Description:        "Details here",
			AcceptanceCriteria: "AC1",
			Complexity:         3,
			OwnedFiles:         []string{"main.go", "util.go"},
			WaveHint:           "parallel",
			DependsOn:          []string{},
		}),
		Timestamp: "2026-01-01T00:01:00Z",
	})

	assertStoryStatus := func(expected string) {
		t.Helper()
		s, err := store.GetStory("story-1")
		if err != nil {
			t.Fatalf("GetStory: %v", err)
		}
		if s.Status != expected {
			t.Errorf("Status: got %q, want %q", s.Status, expected)
		}
	}

	assertStoryStatus("draft")

	// Assign
	store.Project(Event{
		ID:      "evt-2",
		Type:    EventStoryAssigned,
		StoryID: "story-1",
		Payload: mustMarshal(t, StoryAssignedPayload{
			AgentID: "agent-1",
			Wave:    1,
		}),
		Timestamp: "2026-01-01T00:02:00Z",
	})
	assertStoryStatus("assigned")

	// Start
	store.Project(Event{
		ID:        "evt-3",
		Type:      EventStoryStarted,
		StoryID:   "story-1",
		Timestamp: "2026-01-01T00:03:00Z",
	})
	assertStoryStatus("in_progress")

	// Complete (moves to review)
	store.Project(Event{
		ID:        "evt-4",
		Type:      EventStoryCompleted,
		StoryID:   "story-1",
		Timestamp: "2026-01-01T00:04:00Z",
	})
	assertStoryStatus("review")

	// Review passed (moves to qa)
	store.Project(Event{
		ID:        "evt-5",
		Type:      EventStoryReviewPassed,
		StoryID:   "story-1",
		Timestamp: "2026-01-01T00:05:00Z",
	})
	assertStoryStatus("qa")

	// QA passed (moves to pr_submitted)
	store.Project(Event{
		ID:        "evt-6",
		Type:      EventStoryQAPassed,
		StoryID:   "story-1",
		Timestamp: "2026-01-01T00:06:00Z",
	})
	assertStoryStatus("pr_submitted")

	// Merged
	store.Project(Event{
		ID:        "evt-7",
		Type:      EventStoryMerged,
		StoryID:   "story-1",
		Timestamp: "2026-01-01T00:07:00Z",
	})
	assertStoryStatus("merged")
}

func TestSQLiteStore_ListStoriesFiltered(t *testing.T) {
	store := newTestStore(t)

	// Create requirement and two stories
	store.Project(Event{
		ID:   "evt-0",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-1", Title: "R1", Description: "D1", RepoPath: "/r1",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})
	store.Project(Event{
		ID:   "evt-0b",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-2", Title: "R2", Description: "D2", RepoPath: "/r2",
		}),
		Timestamp: "2026-01-01T00:00:01Z",
	})

	store.Project(Event{
		ID: "evt-1", Type: EventStoryCreated, StoryID: "story-1",
		Payload: mustMarshal(t, StoryCreatedPayload{
			ID: "story-1", ReqID: "req-1", Title: "S1", Description: "D", Complexity: 1,
		}),
		Timestamp: "2026-01-01T00:01:00Z",
	})
	store.Project(Event{
		ID: "evt-2", Type: EventStoryCreated, StoryID: "story-2",
		Payload: mustMarshal(t, StoryCreatedPayload{
			ID: "story-2", ReqID: "req-1", Title: "S2", Description: "D", Complexity: 2,
		}),
		Timestamp: "2026-01-01T00:02:00Z",
	})
	store.Project(Event{
		ID: "evt-3", Type: EventStoryCreated, StoryID: "story-3",
		Payload: mustMarshal(t, StoryCreatedPayload{
			ID: "story-3", ReqID: "req-2", Title: "S3", Description: "D", Complexity: 1,
		}),
		Timestamp: "2026-01-01T00:03:00Z",
	})

	// Start story-1 to change its status
	store.Project(Event{
		ID: "evt-4", Type: EventStoryAssigned, StoryID: "story-1",
		Payload: mustMarshal(t, StoryAssignedPayload{AgentID: "a1", Wave: 1}),
		Timestamp: "2026-01-01T00:04:00Z",
	})

	// Filter by req_id
	stories, err := store.ListStories(StoryFilter{ReqID: "req-1"})
	if err != nil {
		t.Fatalf("ListStories by req_id: %v", err)
	}
	if len(stories) != 2 {
		t.Fatalf("expected 2 stories for req-1, got %d", len(stories))
	}

	// Filter by status
	stories, err = store.ListStories(StoryFilter{Status: "assigned"})
	if err != nil {
		t.Fatalf("ListStories by status: %v", err)
	}
	if len(stories) != 1 {
		t.Errorf("expected 1 assigned story, got %d", len(stories))
	}
	if stories[0].ID != "story-1" {
		t.Errorf("expected story-1, got %s", stories[0].ID)
	}

	// Filter by status = draft
	stories, err = store.ListStories(StoryFilter{Status: "draft"})
	if err != nil {
		t.Fatalf("ListStories by draft: %v", err)
	}
	if len(stories) != 2 {
		t.Errorf("expected 2 draft stories, got %d", len(stories))
	}
}

func TestSQLiteStore_ListRequirementsFiltered(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:   "evt-1",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-1", Title: "R1", Description: "D1", RepoPath: "/repos/alpha",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})
	store.Project(Event{
		ID:   "evt-2",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-2", Title: "R2", Description: "D2", RepoPath: "/repos/beta",
		}),
		Timestamp: "2026-01-01T00:01:00Z",
	})
	store.Project(Event{
		ID:   "evt-3",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-3", Title: "R3", Description: "D3", RepoPath: "/repos/alpha",
		}),
		Timestamp: "2026-01-01T00:02:00Z",
	})

	// Archive req-3
	if err := store.ArchiveRequirement("req-3"); err != nil {
		t.Fatalf("ArchiveRequirement: %v", err)
	}

	// Filter by repo_path
	reqs, err := store.ListRequirements(ReqFilter{RepoPath: "/repos/alpha"})
	if err != nil {
		t.Fatalf("ListRequirements by repo_path: %v", err)
	}
	if len(reqs) != 2 {
		t.Fatalf("expected 2 reqs for /repos/alpha, got %d", len(reqs))
	}

	// Filter by repo_path and exclude archived
	reqs, err = store.ListRequirements(ReqFilter{
		RepoPath:        "/repos/alpha",
		ExcludeArchived: true,
	})
	if err != nil {
		t.Fatalf("ListRequirements exclude archived: %v", err)
	}
	if len(reqs) != 1 {
		t.Fatalf("expected 1 non-archived req for /repos/alpha, got %d", len(reqs))
	}
	if reqs[0].ID != "req-1" {
		t.Errorf("expected req-1, got %s", reqs[0].ID)
	}
}

func TestSQLiteStore_ArchiveRequirement(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:   "evt-1",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-1", Title: "R1", Description: "D1", RepoPath: "/repo",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})

	if err := store.ArchiveRequirement("req-1"); err != nil {
		t.Fatalf("ArchiveRequirement: %v", err)
	}

	req, err := store.GetRequirement("req-1")
	if err != nil {
		t.Fatalf("GetRequirement: %v", err)
	}
	if req.Status != "archived" {
		t.Errorf("Status: got %q, want %q", req.Status, "archived")
	}
}

func TestSQLiteStore_ArchiveStoriesByReq(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:   "evt-0",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-1", Title: "R1", Description: "D1", RepoPath: "/repo",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})

	for i, id := range []string{"story-1", "story-2"} {
		store.Project(Event{
			ID: "evt-s" + string(rune('0'+i)), Type: EventStoryCreated, StoryID: id,
			Payload: mustMarshal(t, StoryCreatedPayload{
				ID: id, ReqID: "req-1", Title: "S", Description: "D", Complexity: 1,
			}),
			Timestamp: "2026-01-01T00:01:00Z",
		})
	}

	if err := store.ArchiveStoriesByReq("req-1"); err != nil {
		t.Fatalf("ArchiveStoriesByReq: %v", err)
	}

	stories, err := store.ListStories(StoryFilter{ReqID: "req-1"})
	if err != nil {
		t.Fatalf("ListStories: %v", err)
	}
	for _, s := range stories {
		if s.Status != "archived" {
			t.Errorf("story %s Status: got %q, want %q", s.ID, s.Status, "archived")
		}
	}
}

func TestSQLiteStore_PaginationLimitOffset(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:   "evt-0",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-1", Title: "R", Description: "D", RepoPath: "/r",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})

	for i := range 5 {
		id := "story-" + string(rune('A'+i))
		store.Project(Event{
			ID: "evt-" + id, Type: EventStoryCreated, StoryID: id,
			Payload: mustMarshal(t, StoryCreatedPayload{
				ID: id, ReqID: "req-1", Title: "S" + id, Description: "D", Complexity: 1,
			}),
			Timestamp: "2026-01-01T00:01:00Z",
		})
	}

	// Limit only
	stories, err := store.ListStories(StoryFilter{Limit: 2})
	if err != nil {
		t.Fatalf("ListStories limit: %v", err)
	}
	if len(stories) != 2 {
		t.Errorf("expected 2 stories, got %d", len(stories))
	}

	// Offset + Limit
	stories, err = store.ListStories(StoryFilter{Limit: 2, Offset: 2})
	if err != nil {
		t.Fatalf("ListStories limit+offset: %v", err)
	}
	if len(stories) != 2 {
		t.Errorf("expected 2 stories at offset 2, got %d", len(stories))
	}

	// Offset past end
	stories, err = store.ListStories(StoryFilter{Limit: 2, Offset: 10})
	if err != nil {
		t.Fatalf("ListStories offset past end: %v", err)
	}
	if len(stories) != 0 {
		t.Errorf("expected 0 stories at offset 10, got %d", len(stories))
	}
}

func TestSQLiteStore_ListAgents(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:      "evt-1",
		Type:    EventAgentSpawned,
		AgentID: "agent-1",
		Payload: mustMarshal(t, AgentSpawnedPayload{
			ID:          "agent-1",
			Type:        "coder",
			Model:       "claude-sonnet",
			Runtime:     "tmux",
			SessionName: "sess-1",
			StoryID:     "story-1",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})
	store.Project(Event{
		ID:      "evt-2",
		Type:    EventAgentSpawned,
		AgentID: "agent-2",
		Payload: mustMarshal(t, AgentSpawnedPayload{
			ID:          "agent-2",
			Type:        "reviewer",
			Model:       "claude-opus",
			Runtime:     "tmux",
			SessionName: "sess-2",
			StoryID:     "story-2",
		}),
		Timestamp: "2026-01-01T00:01:00Z",
	})

	agents, err := store.ListAgents(AgentFilter{})
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}

	// Filter by status (spawned agents should be "active")
	agents, err = store.ListAgents(AgentFilter{Status: "active"})
	if err != nil {
		t.Fatalf("ListAgents active: %v", err)
	}
	if len(agents) != 2 {
		t.Errorf("expected 2 active agents, got %d", len(agents))
	}

	// No agents with unknown status
	agents, err = store.ListAgents(AgentFilter{Status: "dead"})
	if err != nil {
		t.Fatalf("ListAgents dead: %v", err)
	}
	if len(agents) != 0 {
		t.Errorf("expected 0 dead agents, got %d", len(agents))
	}
}

func TestSQLiteStore_ListEscalations(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:      "evt-1",
		Type:    EventEscalationCreated,
		StoryID: "story-1",
		Payload: mustMarshal(t, EscalationCreatedPayload{
			ID:        "esc-1",
			StoryID:   "story-1",
			FromAgent: "agent-1",
			Reason:    "Build failed",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})
	store.Project(Event{
		ID:      "evt-2",
		Type:    EventEscalationCreated,
		StoryID: "story-2",
		Payload: mustMarshal(t, EscalationCreatedPayload{
			ID:        "esc-2",
			StoryID:   "story-2",
			FromAgent: "agent-2",
			Reason:    "Tests failing",
		}),
		Timestamp: "2026-01-01T00:01:00Z",
	})

	escs, err := store.ListEscalations()
	if err != nil {
		t.Fatalf("ListEscalations: %v", err)
	}
	if len(escs) != 2 {
		t.Fatalf("expected 2 escalations, got %d", len(escs))
	}

	// Should be ordered DESC by created_at, so esc-2 first
	if escs[0].ID != "esc-2" {
		t.Errorf("first escalation: got %q, want %q", escs[0].ID, "esc-2")
	}
	if escs[1].ID != "esc-1" {
		t.Errorf("second escalation: got %q, want %q", escs[1].ID, "esc-1")
	}
	if escs[0].Reason != "Tests failing" {
		t.Errorf("Reason: got %q, want %q", escs[0].Reason, "Tests failing")
	}
}

func TestSQLiteStore_ListStoryDeps(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:   "evt-0",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-1", Title: "R", Description: "D", RepoPath: "/r",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})

	// Story-B depends on Story-A
	store.Project(Event{
		ID: "evt-1", Type: EventStoryCreated, StoryID: "story-A",
		Payload: mustMarshal(t, StoryCreatedPayload{
			ID: "story-A", ReqID: "req-1", Title: "SA", Description: "D",
			Complexity: 1, DependsOn: []string{},
		}),
		Timestamp: "2026-01-01T00:01:00Z",
	})
	store.Project(Event{
		ID: "evt-2", Type: EventStoryCreated, StoryID: "story-B",
		Payload: mustMarshal(t, StoryCreatedPayload{
			ID: "story-B", ReqID: "req-1", Title: "SB", Description: "D",
			Complexity: 2, DependsOn: []string{"story-A"},
		}),
		Timestamp: "2026-01-01T00:02:00Z",
	})

	deps, err := store.ListStoryDeps("req-1")
	if err != nil {
		t.Fatalf("ListStoryDeps: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("expected 1 dep, got %d", len(deps))
	}
	if deps[0].StoryID != "story-B" {
		t.Errorf("StoryID: got %q, want %q", deps[0].StoryID, "story-B")
	}
	if deps[0].DependsOnID != "story-A" {
		t.Errorf("DependsOnID: got %q, want %q", deps[0].DependsOnID, "story-A")
	}
}

func TestSQLiteStore_GetStory_OwnedFiles(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:   "evt-0",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-1", Title: "R", Description: "D", RepoPath: "/r",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})

	expected := []string{"internal/api/handler.go", "internal/api/routes.go", "pkg/util/helpers.go"}
	store.Project(Event{
		ID: "evt-1", Type: EventStoryCreated, StoryID: "story-1",
		Payload: mustMarshal(t, StoryCreatedPayload{
			ID: "story-1", ReqID: "req-1", Title: "S", Description: "D",
			Complexity: 2, OwnedFiles: expected, WaveHint: "sequential",
		}),
		Timestamp: "2026-01-01T00:01:00Z",
	})

	story, err := store.GetStory("story-1")
	if err != nil {
		t.Fatalf("GetStory: %v", err)
	}

	if len(story.OwnedFiles) != len(expected) {
		t.Fatalf("OwnedFiles length: got %d, want %d", len(story.OwnedFiles), len(expected))
	}
	for i, f := range expected {
		if story.OwnedFiles[i] != f {
			t.Errorf("OwnedFiles[%d]: got %q, want %q", i, story.OwnedFiles[i], f)
		}
	}
	if story.WaveHint != "sequential" {
		t.Errorf("WaveHint: got %q, want %q", story.WaveHint, "sequential")
	}
}

func TestSQLiteStore_StoryPRCreated(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:   "evt-0",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-1", Title: "R", Description: "D", RepoPath: "/r",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})

	store.Project(Event{
		ID: "evt-1", Type: EventStoryCreated, StoryID: "story-1",
		Payload: mustMarshal(t, StoryCreatedPayload{
			ID: "story-1", ReqID: "req-1", Title: "S", Description: "D", Complexity: 1,
		}),
		Timestamp: "2026-01-01T00:01:00Z",
	})

	store.Project(Event{
		ID: "evt-2", Type: EventStoryPRCreated, StoryID: "story-1",
		Payload: mustMarshal(t, StoryPRCreatedPayload{
			PRUrl:    "https://github.com/org/repo/pull/42",
			PRNumber: 42,
		}),
		Timestamp: "2026-01-01T00:02:00Z",
	})

	story, err := store.GetStory("story-1")
	if err != nil {
		t.Fatalf("GetStory: %v", err)
	}
	if story.PRUrl != "https://github.com/org/repo/pull/42" {
		t.Errorf("PRUrl: got %q, want %q", story.PRUrl, "https://github.com/org/repo/pull/42")
	}
	if story.PRNumber != 42 {
		t.Errorf("PRNumber: got %d, want %d", story.PRNumber, 42)
	}
	if story.Status != "pr_submitted" {
		t.Errorf("Status: got %q, want %q", story.Status, "pr_submitted")
	}
}

func TestSQLiteStore_ReviewFailed(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:   "evt-0",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-1", Title: "R", Description: "D", RepoPath: "/r",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})

	store.Project(Event{
		ID: "evt-1", Type: EventStoryCreated, StoryID: "story-1",
		Payload: mustMarshal(t, StoryCreatedPayload{
			ID: "story-1", ReqID: "req-1", Title: "S", Description: "D", Complexity: 1,
		}),
		Timestamp: "2026-01-01T00:01:00Z",
	})

	// Complete story, then review fails
	store.Project(Event{
		ID: "evt-2", Type: EventStoryCompleted, StoryID: "story-1",
		Timestamp: "2026-01-01T00:02:00Z",
	})
	store.Project(Event{
		ID: "evt-3", Type: EventStoryReviewFailed, StoryID: "story-1",
		Timestamp: "2026-01-01T00:03:00Z",
	})

	story, err := store.GetStory("story-1")
	if err != nil {
		t.Fatalf("GetStory: %v", err)
	}
	if story.Status != "draft" {
		t.Errorf("Status after review failed: got %q, want %q", story.Status, "draft")
	}
}

func TestSQLiteStore_QAFailed(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:   "evt-0",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-1", Title: "R", Description: "D", RepoPath: "/r",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})

	store.Project(Event{
		ID: "evt-1", Type: EventStoryCreated, StoryID: "story-1",
		Payload: mustMarshal(t, StoryCreatedPayload{
			ID: "story-1", ReqID: "req-1", Title: "S", Description: "D", Complexity: 1,
		}),
		Timestamp: "2026-01-01T00:01:00Z",
	})

	store.Project(Event{
		ID: "evt-2", Type: EventStoryCompleted, StoryID: "story-1",
		Timestamp: "2026-01-01T00:02:00Z",
	})
	store.Project(Event{
		ID: "evt-3", Type: EventStoryReviewPassed, StoryID: "story-1",
		Timestamp: "2026-01-01T00:03:00Z",
	})
	store.Project(Event{
		ID: "evt-4", Type: EventStoryQAFailed, StoryID: "story-1",
		Timestamp: "2026-01-01T00:04:00Z",
	})

	story, err := store.GetStory("story-1")
	if err != nil {
		t.Fatalf("GetStory: %v", err)
	}
	if story.Status != "qa_failed" {
		t.Errorf("Status after QA failed: got %q, want %q", story.Status, "qa_failed")
	}
}

func TestSQLiteStore_ReqStatusTransitions(t *testing.T) {
	store := newTestStore(t)

	store.Project(Event{
		ID:   "evt-1",
		Type: EventReqSubmitted,
		Payload: mustMarshal(t, ReqSubmittedPayload{
			ID: "req-1", Title: "R", Description: "D", RepoPath: "/r",
		}),
		Timestamp: "2026-01-01T00:00:00Z",
	})

	transitions := []struct {
		evtType  EventType
		expected string
	}{
		{EventReqAnalyzed, "analyzed"},
		{EventReqPlanned, "planned"},
		{EventReqPaused, "paused"},
		{EventReqResumed, "resumed"},
		{EventReqCompleted, "completed"},
	}

	for i, tc := range transitions {
		store.Project(Event{
			ID:      "evt-" + string(rune('A'+i)),
			Type:    tc.evtType,
			StoryID: "req-1",
			Payload: mustMarshal(t, map[string]string{"req_id": "req-1"}),
			Timestamp: "2026-01-01T00:0" + string(rune('1'+i)) + ":00Z",
		})

		req, err := store.GetRequirement("req-1")
		if err != nil {
			t.Fatalf("GetRequirement after %s: %v", tc.evtType, err)
		}
		if req.Status != tc.expected {
			t.Errorf("after %s: got %q, want %q", tc.evtType, req.Status, tc.expected)
		}
	}
}

func TestSQLiteStore_UnknownEventIgnored(t *testing.T) {
	store := newTestStore(t)

	err := store.Project(Event{
		ID:        "evt-1",
		Type:      EventType("future.event"),
		Timestamp: "2026-01-01T00:00:00Z",
	})
	if err != nil {
		t.Errorf("unknown event should not error, got: %v", err)
	}
}
