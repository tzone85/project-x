package planner

import (
	"strings"
	"testing"

	"github.com/tzone85/project-x/internal/config"
)

func TestBuildDecompositionMessagesStructure(t *testing.T) {
	req := Requirement{ID: "REQ-1", Title: "Test Feature", Description: "Build something"}
	ts := testTechStack()
	cfg := defaultPlanningConfig()

	msgs := BuildDecompositionMessages(req, ts, cfg)

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages (system + user), got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("expected first message role=system, got %s", msgs[0].Role)
	}
	if msgs[1].Role != "user" {
		t.Errorf("expected second message role=user, got %s", msgs[1].Role)
	}
}

func TestBuildDecompositionIncludesTechStack(t *testing.T) {
	req := Requirement{ID: "REQ-1", Title: "Test", Description: "Desc"}
	ts := TechStack{Language: "Go", Framework: "Cobra", TestRunner: "go test"}
	cfg := defaultPlanningConfig()

	msgs := BuildDecompositionMessages(req, ts, cfg)

	if !strings.Contains(msgs[0].Content, "Go") {
		t.Error("expected system prompt to contain language 'Go'")
	}
	if !strings.Contains(msgs[0].Content, "Cobra") {
		t.Error("expected system prompt to contain framework 'Cobra'")
	}
	if !strings.Contains(msgs[0].Content, "go test") {
		t.Error("expected system prompt to contain test runner 'go test'")
	}
}

func TestBuildDecompositionOmitsEmptyTechStack(t *testing.T) {
	req := Requirement{ID: "REQ-1", Title: "Test", Description: "Desc"}
	ts := TechStack{} // empty
	cfg := defaultPlanningConfig()

	msgs := BuildDecompositionMessages(req, ts, cfg)

	if strings.Contains(msgs[0].Content, "Tech Stack") {
		t.Error("expected no Tech Stack section for empty tech stack")
	}
}

func TestBuildDecompositionIncludesConstraints(t *testing.T) {
	req := Requirement{ID: "REQ-1", Title: "Test", Description: "Desc"}
	ts := testTechStack()
	cfg := defaultPlanningConfig()

	msgs := BuildDecompositionMessages(req, ts, cfg)

	if !strings.Contains(msgs[0].Content, "Maximum story complexity: 8") {
		t.Error("expected system prompt to contain max complexity constraint")
	}
	if !strings.Contains(msgs[0].Content, "Maximum stories per requirement: 15") {
		t.Error("expected system prompt to contain max stories constraint")
	}
}

func TestBuildDecompositionIncludesRequirement(t *testing.T) {
	req := Requirement{ID: "REQ-42", Title: "Auth System", Description: "Build OAuth2 flow"}
	ts := testTechStack()
	cfg := defaultPlanningConfig()

	msgs := BuildDecompositionMessages(req, ts, cfg)

	if !strings.Contains(msgs[1].Content, "REQ-42") {
		t.Error("expected user message to contain requirement ID")
	}
	if !strings.Contains(msgs[1].Content, "Auth System") {
		t.Error("expected user message to contain requirement title")
	}
	if !strings.Contains(msgs[1].Content, "Build OAuth2 flow") {
		t.Error("expected user message to contain requirement description")
	}
}

func TestBuildValidationMessagesStructure(t *testing.T) {
	req := Requirement{ID: "REQ-1", Title: "Test", Description: "Desc"}
	stories := []Story{validStory("STR-1")}
	cfg := defaultPlanningConfig()

	msgs := BuildValidationMessages(req, stories, cfg)

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].Role != "system" {
		t.Errorf("expected system role, got %s", msgs[0].Role)
	}
	if !strings.Contains(msgs[0].Content, "QA reviewer") {
		t.Error("expected validation system prompt to mention QA reviewer")
	}
}

func TestBuildValidationIncludesStories(t *testing.T) {
	req := Requirement{ID: "REQ-1", Title: "Test", Description: "Desc"}
	stories := []Story{validStory("STR-1")}
	cfg := defaultPlanningConfig()

	msgs := BuildValidationMessages(req, stories, cfg)

	if !strings.Contains(msgs[1].Content, "STR-1") {
		t.Error("expected user message to contain story ID")
	}
}

func TestBuildRefinementMessagesStructure(t *testing.T) {
	req := Requirement{ID: "REQ-1", Title: "Test", Description: "Desc"}
	ts := testTechStack()
	cfg := defaultPlanningConfig()
	critique := "Stories need better acceptance criteria"

	msgs := BuildRefinementMessages(req, ts, cfg, critique)

	if len(msgs) != 4 {
		t.Fatalf("expected 4 messages (system + user + assistant + feedback), got %d", len(msgs))
	}
	if msgs[2].Role != "assistant" {
		t.Errorf("expected third message role=assistant, got %s", msgs[2].Role)
	}
	if !strings.Contains(msgs[3].Content, critique) {
		t.Error("expected feedback message to contain critique")
	}
}

func TestBuildDecompositionFileOwnershipEnforced(t *testing.T) {
	req := Requirement{ID: "REQ-1", Title: "Test", Description: "Desc"}
	ts := testTechStack()
	cfg := defaultPlanningConfig()
	cfg.EnforceFileOwnership = true

	msgs := BuildDecompositionMessages(req, ts, cfg)

	if !strings.Contains(msgs[0].Content, "No two stories may own the same file") {
		t.Error("expected file ownership constraint when enforced")
	}
}

func TestBuildDecompositionFileOwnershipNotEnforced(t *testing.T) {
	req := Requirement{ID: "REQ-1", Title: "Test", Description: "Desc"}
	ts := testTechStack()
	cfg := config.PlanningConfig{
		RequiredFields:          []string{"title"},
		MaxStoryComplexity:      8,
		MaxStoriesPerRequirement: 15,
		EnforceFileOwnership:    false,
	}

	msgs := BuildDecompositionMessages(req, ts, cfg)

	if strings.Contains(msgs[0].Content, "No two stories may own the same file") {
		t.Error("should not mention file ownership when not enforced")
	}
}
