package agent

import (
	"strings"
	"testing"
)

func TestGenerateSystemPrompt(t *testing.T) {
	tests := []struct {
		name     string
		role     Role
		contains []string
	}{
		{
			name: "junior_prompt",
			role: RoleJunior,
			contains: []string{
				"Junior Developer",
				"implement",
				"established patterns",
			},
		},
		{
			name: "senior_prompt",
			role: RoleSenior,
			contains: []string{
				"Senior Developer",
				"review",
				"plan",
			},
		},
		{
			name: "tech_lead_prompt",
			role: RoleTechLead,
			contains: []string{
				"Tech Lead",
				"review",
				"merge",
				"audit",
			},
		},
		{
			name: "qa_prompt",
			role: RoleQA,
			contains: []string{
				"QA Engineer",
				"test",
			},
		},
		{
			name: "auditor_prompt",
			role: RoleAuditor,
			contains: []string{
				"Code Auditor",
				"review",
				"audit",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt, err := GenerateSystemPrompt(tt.role)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if prompt == "" {
				t.Fatal("expected non-empty prompt")
			}
			for _, substr := range tt.contains {
				if !strings.Contains(strings.ToLower(prompt), strings.ToLower(substr)) {
					t.Errorf("prompt missing expected content %q", substr)
				}
			}
		})
	}
}

func TestGenerateSystemPromptUnknownRole(t *testing.T) {
	_, err := GenerateSystemPrompt(Role("nonexistent"))
	if err == nil {
		t.Fatal("expected error for unknown role")
	}
}

func TestGenerateStoryPrompt(t *testing.T) {
	ctx := StoryContext{
		StoryID:     "STR-001",
		Title:       "Add user login endpoint",
		Description: "Implement POST /api/login with JWT tokens",
		AcceptanceCriteria: []string{
			"Returns JWT on valid credentials",
			"Returns 401 on invalid credentials",
			"Rate-limits to 5 attempts per minute",
		},
		OwnedFiles: []string{"internal/auth/login.go", "internal/auth/login_test.go"},
		Complexity: 3,
	}

	prompt, err := GenerateStoryPrompt(RoleJunior, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedContains := []string{
		"STR-001",
		"Add user login endpoint",
		"POST /api/login",
		"JWT on valid credentials",
		"401 on invalid credentials",
		"Rate-limits",
		"internal/auth/login.go",
	}

	for _, substr := range expectedContains {
		if !strings.Contains(prompt, substr) {
			t.Errorf("story prompt missing %q", substr)
		}
	}
}

func TestGenerateStoryPromptWithTechStack(t *testing.T) {
	ctx := StoryContext{
		StoryID:            "STR-002",
		Title:              "Add database migration",
		Description:        "Create migration for users table",
		AcceptanceCriteria: []string{"Table created successfully"},
		Complexity:         2,
		TechStack: &TechStack{
			Language:    "Go",
			Framework:   "stdlib",
			TestRunner:  "go test",
			BuildTool:   "go build",
			Linter:      "golangci-lint",
			PackageJSON: false,
		},
	}

	prompt, err := GenerateStoryPrompt(RoleJunior, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedContains := []string{
		"Go",
		"go test",
		"golangci-lint",
	}

	for _, substr := range expectedContains {
		if !strings.Contains(prompt, substr) {
			t.Errorf("story prompt with tech stack missing %q", substr)
		}
	}
}

func TestGenerateStoryPromptWithoutTechStack(t *testing.T) {
	ctx := StoryContext{
		StoryID:            "STR-003",
		Title:              "Simple fix",
		Description:        "Fix a typo",
		AcceptanceCriteria: []string{"Typo fixed"},
		Complexity:         1,
	}

	prompt, err := GenerateStoryPrompt(RoleJunior, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(prompt, "Tech Stack") {
		t.Error("prompt should not contain Tech Stack section when no tech stack provided")
	}
}

func TestGenerateStoryPromptUnknownRole(t *testing.T) {
	ctx := StoryContext{
		StoryID:            "STR-001",
		Title:              "Test",
		AcceptanceCriteria: []string{"Done"},
		Complexity:         1,
	}

	_, err := GenerateStoryPrompt(Role("nonexistent"), ctx)
	if err == nil {
		t.Fatal("expected error for unknown role")
	}
}

func TestTechStackString(t *testing.T) {
	ts := TechStack{
		Language:   "Go",
		Framework:  "stdlib",
		TestRunner: "go test",
		BuildTool:  "go build",
		Linter:     "golangci-lint",
	}

	s := ts.String()

	if !strings.Contains(s, "Go") {
		t.Error("missing language")
	}
	if !strings.Contains(s, "go test") {
		t.Error("missing test runner")
	}
}

func TestGenerateStoryPromptEmptyAcceptanceCriteria(t *testing.T) {
	ctx := StoryContext{
		StoryID:            "STR-004",
		Title:              "No criteria",
		Description:        "Missing AC",
		AcceptanceCriteria: []string{},
		Complexity:         1,
	}

	prompt, err := GenerateStoryPrompt(RoleJunior, ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if prompt == "" {
		t.Fatal("expected non-empty prompt even with empty AC")
	}
}
