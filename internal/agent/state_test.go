package agent

import (
	"testing"
)

func TestNewAgent(t *testing.T) {
	a := NewAgent("agent-1", RoleJunior)

	if a.ID != "agent-1" {
		t.Errorf("got ID %q, want %q", a.ID, "agent-1")
	}
	if a.Role != RoleJunior {
		t.Errorf("got Role %q, want %q", a.Role, RoleJunior)
	}
	if a.Status != StatusIdle {
		t.Errorf("got Status %q, want %q", a.Status, StatusIdle)
	}
	if a.StoryID != "" {
		t.Errorf("expected empty StoryID, got %q", a.StoryID)
	}
}

func TestAgentStatusValues(t *testing.T) {
	tests := []struct {
		status Status
		want   string
	}{
		{StatusIdle, "idle"},
		{StatusWorking, "working"},
		{StatusBlocked, "blocked"},
		{StatusTerminated, "terminated"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := string(tt.status); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
	}{
		{"idle_to_working", StatusIdle, StatusWorking},
		{"working_to_blocked", StatusWorking, StatusBlocked},
		{"working_to_idle", StatusWorking, StatusIdle},
		{"working_to_terminated", StatusWorking, StatusTerminated},
		{"blocked_to_working", StatusBlocked, StatusWorking},
		{"blocked_to_terminated", StatusBlocked, StatusTerminated},
		{"idle_to_terminated", StatusIdle, StatusTerminated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAgent("test", RoleJunior)
			a = a.withStatus(tt.from)

			result, err := a.TransitionTo(tt.to)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Status != tt.to {
				t.Errorf("got status %q, want %q", result.Status, tt.to)
			}
			// Verify original is unchanged (immutability)
			if a.Status != tt.from {
				t.Errorf("original agent was mutated: got %q, want %q", a.Status, tt.from)
			}
		})
	}
}

func TestInvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from Status
		to   Status
	}{
		{"idle_to_blocked", StatusIdle, StatusBlocked},
		{"terminated_to_idle", StatusTerminated, StatusIdle},
		{"terminated_to_working", StatusTerminated, StatusWorking},
		{"terminated_to_blocked", StatusTerminated, StatusBlocked},
		{"idle_to_idle", StatusIdle, StatusIdle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := NewAgent("test", RoleJunior)
			a = a.withStatus(tt.from)

			_, err := a.TransitionTo(tt.to)
			if err == nil {
				t.Fatalf("expected error for transition %s -> %s", tt.from, tt.to)
			}
		})
	}
}

func TestAssignStory(t *testing.T) {
	a := NewAgent("agent-1", RoleSenior)

	result, err := a.AssignStory("STR-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StoryID != "STR-001" {
		t.Errorf("got StoryID %q, want %q", result.StoryID, "STR-001")
	}
	if result.Status != StatusWorking {
		t.Errorf("got Status %q, want %q", result.Status, StatusWorking)
	}
	// Original unchanged
	if a.StoryID != "" {
		t.Error("original agent was mutated")
	}
}

func TestAssignStoryWhenNotIdle(t *testing.T) {
	a := NewAgent("agent-1", RoleSenior)
	a = a.withStatus(StatusWorking)

	_, err := a.AssignStory("STR-002")
	if err == nil {
		t.Fatal("expected error when assigning story to non-idle agent")
	}
}

func TestCompleteStory(t *testing.T) {
	a := NewAgent("agent-1", RoleSenior)
	a, _ = a.AssignStory("STR-001")

	result, err := a.CompleteStory()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.StoryID != "" {
		t.Errorf("expected empty StoryID after completion, got %q", result.StoryID)
	}
	if result.Status != StatusIdle {
		t.Errorf("got Status %q, want %q", result.Status, StatusIdle)
	}
}

func TestCompleteStoryWhenNotWorking(t *testing.T) {
	a := NewAgent("agent-1", RoleSenior)

	_, err := a.CompleteStory()
	if err == nil {
		t.Fatal("expected error when completing story from non-working state")
	}
}

func TestAgentMemory(t *testing.T) {
	a := NewAgent("agent-1", RoleJunior)

	if len(a.Memory) != 0 {
		t.Errorf("expected empty memory, got %d entries", len(a.Memory))
	}

	a = a.WithMemory("context", "some important context")

	if a.Memory["context"] != "some important context" {
		t.Errorf("got memory %q, want %q", a.Memory["context"], "some important context")
	}
}

func TestAgentMemoryImmutability(t *testing.T) {
	a := NewAgent("agent-1", RoleJunior)
	b := a.WithMemory("key1", "value1")
	c := b.WithMemory("key2", "value2")

	if len(a.Memory) != 0 {
		t.Error("original agent memory was mutated")
	}
	if len(b.Memory) != 1 {
		t.Errorf("expected 1 memory entry in b, got %d", len(b.Memory))
	}
	if len(c.Memory) != 2 {
		t.Errorf("expected 2 memory entries in c, got %d", len(c.Memory))
	}
}

func TestParseStatusValid(t *testing.T) {
	tests := []struct {
		input string
		want  Status
	}{
		{"idle", StatusIdle},
		{"working", StatusWorking},
		{"blocked", StatusBlocked},
		{"terminated", StatusTerminated},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseStatus(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseStatusInvalid(t *testing.T) {
	_, err := ParseStatus("unknown")
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}
