package agent

import (
	"strings"
	"testing"
)

// --- Role tests ---

func TestAllRolesReturnsSevenRoles(t *testing.T) {
	roles := AllRoles()
	if len(roles) != 7 {
		t.Errorf("got %d roles, want 7", len(roles))
	}
}

func TestDefaultRoleDefinitionsComplete(t *testing.T) {
	defs := DefaultRoleDefinitions()
	for _, role := range AllRoles() {
		if _, ok := defs[role]; !ok {
			t.Errorf("missing definition for role %q", role)
		}
	}
}

func TestCanHandle(t *testing.T) {
	defs := DefaultRoleDefinitions()

	tests := []struct {
		role       Role
		complexity int
		want       bool
	}{
		{RoleJunior, 2, true},
		{RoleJunior, 3, true},
		{RoleJunior, 4, false},
		{RoleIntermediate, 5, true},
		{RoleIntermediate, 6, false},
		{RoleSenior, 8, true},
		{RoleTechLead, 8, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.role), func(t *testing.T) {
			rd := defs[tt.role]
			if got := rd.CanHandle(tt.complexity); got != tt.want {
				t.Errorf("CanHandle(%d) = %v, want %v", tt.complexity, got, tt.want)
			}
		})
	}
}

// --- Scoring tests ---

func TestRoleForComplexity(t *testing.T) {
	defs := DefaultRoleDefinitions()

	tests := []struct {
		complexity int
		want       Role
	}{
		{1, RoleJunior},
		{3, RoleJunior},
		{4, RoleIntermediate},
		{5, RoleIntermediate},
		{6, RoleSenior},
		{8, RoleSenior},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := RoleForComplexity(tt.complexity, defs)
			if got != tt.want {
				t.Errorf("RoleForComplexity(%d) = %q, want %q", tt.complexity, got, tt.want)
			}
		})
	}
}

// --- Prompt tests ---

func TestSystemPromptIncludesRoleInfo(t *testing.T) {
	pb := NewPromptBuilder(DefaultRoleDefinitions())
	prompt := pb.SystemPrompt(RoleSenior, TechStack{Language: "Go", TestRunner: "go test"})

	if !strings.Contains(prompt, "Senior Developer") {
		t.Error("system prompt missing role display name")
	}
	if !strings.Contains(prompt, "Go") {
		t.Error("system prompt missing language from tech stack")
	}
	if !strings.Contains(prompt, "go test") {
		t.Error("system prompt missing test runner")
	}
}

func TestSystemPromptUnknownRole(t *testing.T) {
	pb := NewPromptBuilder(DefaultRoleDefinitions())
	prompt := pb.SystemPrompt(Role("unknown"), TechStack{})

	if !strings.Contains(prompt, "unknown") {
		t.Error("expected fallback prompt mentioning the unknown role")
	}
}

func TestSystemPromptEmptyTechStack(t *testing.T) {
	pb := NewPromptBuilder(DefaultRoleDefinitions())
	prompt := pb.SystemPrompt(RoleSenior, TechStack{})

	if strings.Contains(prompt, "Tech Stack") {
		t.Error("should not include Tech Stack section when empty")
	}
}

func TestSystemPromptIncludesConstraints(t *testing.T) {
	pb := NewPromptBuilder(DefaultRoleDefinitions())
	prompt := pb.SystemPrompt(RoleTechLead, TechStack{})

	if !strings.Contains(prompt, "does not implement stories directly") {
		t.Error("system prompt missing constraints")
	}
}

func TestStoryPromptIncludesAllFields(t *testing.T) {
	pb := NewPromptBuilder(DefaultRoleDefinitions())
	prompt := pb.StoryPrompt(StorySpec{
		ID:                 "STR-001",
		Title:              "Add login feature",
		Description:        "Implement OAuth2 login flow",
		AcceptanceCriteria: []string{"User can log in", "Session persists"},
		OwnedFiles:         []string{"auth/login.go", "auth/session.go"},
		Complexity:         5,
		DependsOn:          []string{"STR-000"},
	})

	checks := []string{
		"STR-001",
		"Add login feature",
		"OAuth2 login flow",
		"User can log in",
		"auth/login.go",
		"STR-000",
		"5",
	}
	for _, check := range checks {
		if !strings.Contains(prompt, check) {
			t.Errorf("story prompt missing %q", check)
		}
	}
}

func TestStoryPromptMinimalFields(t *testing.T) {
	pb := NewPromptBuilder(DefaultRoleDefinitions())
	prompt := pb.StoryPrompt(StorySpec{
		ID:    "STR-002",
		Title: "Simple task",
	})

	if !strings.Contains(prompt, "STR-002") {
		t.Error("prompt missing story ID")
	}
	if strings.Contains(prompt, "Dependencies") {
		t.Error("should not include Dependencies section when empty")
	}
	if strings.Contains(prompt, "Owned Files") {
		t.Error("should not include Owned Files section when empty")
	}
}

// --- State tests ---

func TestStateManagerRegisterAndGet(t *testing.T) {
	sm := NewStateManager()
	state := sm.Register("agent-1", RoleSenior)

	if state.Status != StatusIdle {
		t.Errorf("status = %q, want %q", state.Status, StatusIdle)
	}
	if state.Role != RoleSenior {
		t.Errorf("role = %q, want %q", state.Role, RoleSenior)
	}

	got, ok := sm.Get("agent-1")
	if !ok {
		t.Fatal("agent not found")
	}
	if got.SessionName != "agent-1" {
		t.Errorf("session = %q, want %q", got.SessionName, "agent-1")
	}
}

func TestStateManagerGetNotFound(t *testing.T) {
	sm := NewStateManager()
	_, ok := sm.Get("nonexistent")
	if ok {
		t.Error("expected not found")
	}
}

func TestStateManagerAssignStory(t *testing.T) {
	sm := NewStateManager()
	sm.Register("agent-1", RoleSenior)

	state, err := sm.AssignStory("agent-1", "story-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Status != StatusWorking {
		t.Errorf("status = %q, want %q", state.Status, StatusWorking)
	}
	if state.CurrentStory != "story-1" {
		t.Errorf("current story = %q, want %q", state.CurrentStory, "story-1")
	}
}

func TestStateManagerAssignStoryNotIdle(t *testing.T) {
	sm := NewStateManager()
	sm.Register("agent-1", RoleSenior)
	sm.AssignStory("agent-1", "story-1")

	_, err := sm.AssignStory("agent-1", "story-2")
	if err == nil {
		t.Fatal("expected error assigning to non-idle agent")
	}
}

func TestStateManagerAssignStoryNotFound(t *testing.T) {
	sm := NewStateManager()
	_, err := sm.AssignStory("nonexistent", "story-1")
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}

func TestStateManagerReleaseStory(t *testing.T) {
	sm := NewStateManager()
	sm.Register("agent-1", RoleSenior)
	sm.AssignStory("agent-1", "story-1")

	state, err := sm.ReleaseStory("agent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Status != StatusIdle {
		t.Errorf("status = %q, want %q", state.Status, StatusIdle)
	}
	if state.CurrentStory != "" {
		t.Errorf("current story = %q, want empty", state.CurrentStory)
	}
}

func TestStateManagerSetStatus(t *testing.T) {
	sm := NewStateManager()
	sm.Register("agent-1", RoleSenior)
	sm.AssignStory("agent-1", "story-1")

	state, err := sm.SetStatus("agent-1", StatusBlocked)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Status != StatusBlocked {
		t.Errorf("status = %q, want %q", state.Status, StatusBlocked)
	}
	// Story should still be assigned when blocked
	if state.CurrentStory != "story-1" {
		t.Errorf("current story = %q, want %q", state.CurrentStory, "story-1")
	}
}

func TestStateManagerTerminatedClearsStory(t *testing.T) {
	sm := NewStateManager()
	sm.Register("agent-1", RoleSenior)
	sm.AssignStory("agent-1", "story-1")

	state, _ := sm.SetStatus("agent-1", StatusTerminated)
	if state.CurrentStory != "" {
		t.Errorf("terminated agent should have no story, got %q", state.CurrentStory)
	}
}

func TestStateManagerListByStatus(t *testing.T) {
	sm := NewStateManager()
	sm.Register("agent-1", RoleSenior)
	sm.Register("agent-2", RoleJunior)
	sm.Register("agent-3", RoleSenior)
	sm.AssignStory("agent-1", "story-1")

	idle := sm.ListByStatus(StatusIdle)
	if len(idle) != 2 {
		t.Errorf("idle agents = %d, want 2", len(idle))
	}

	working := sm.ListByStatus(StatusWorking)
	if len(working) != 1 {
		t.Errorf("working agents = %d, want 1", len(working))
	}
}

func TestStateManagerAll(t *testing.T) {
	sm := NewStateManager()
	sm.Register("a1", RoleSenior)
	sm.Register("a2", RoleJunior)

	all := sm.All()
	if len(all) != 2 {
		t.Errorf("all agents = %d, want 2", len(all))
	}
}

func TestSetStatusNotFound(t *testing.T) {
	sm := NewStateManager()
	_, err := sm.SetStatus("nonexistent", StatusBlocked)
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}

func TestReleaseStoryNotFound(t *testing.T) {
	sm := NewStateManager()
	_, err := sm.ReleaseStory("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent agent")
	}
}
