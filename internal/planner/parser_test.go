package planner

import (
	"testing"
)

func TestParseStoriesValidJSON(t *testing.T) {
	raw := `[{"id":"STR-1","title":"Setup","description":"Setup project","acceptance_criteria":["works"],"owned_files":["main.go"],"complexity":2,"depends_on":[]}]`

	stories, err := ParseStories(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(stories))
	}
	if stories[0].ID != "STR-1" {
		t.Errorf("expected ID=STR-1, got %s", stories[0].ID)
	}
	if stories[0].Complexity != 2 {
		t.Errorf("expected Complexity=2, got %d", stories[0].Complexity)
	}
}

func TestParseStoriesWithCodeFences(t *testing.T) {
	raw := "```json\n" +
		`[{"id":"STR-1","title":"Setup","description":"desc","acceptance_criteria":["ac"],"owned_files":["f.go"],"complexity":1,"depends_on":[]}]` +
		"\n```"

	stories, err := ParseStories(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(stories))
	}
	if stories[0].Title != "Setup" {
		t.Errorf("expected Title=Setup, got %s", stories[0].Title)
	}
}

func TestParseStoriesWithPlainCodeFences(t *testing.T) {
	raw := "```\n" +
		`[{"id":"STR-1","title":"T","description":"D","acceptance_criteria":["a"],"owned_files":["f"],"complexity":1,"depends_on":[]}]` +
		"\n```"

	stories, err := ParseStories(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stories) != 1 {
		t.Fatalf("expected 1 story, got %d", len(stories))
	}
}

func TestParseStoriesEmptyArray(t *testing.T) {
	_, err := ParseStories("[]")
	if err == nil {
		t.Error("expected error for empty stories array")
	}
}

func TestParseStoriesInvalidJSON(t *testing.T) {
	_, err := ParseStories("not json at all")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseStoriesMultiple(t *testing.T) {
	raw := `[
		{"id":"STR-1","title":"A","description":"D","acceptance_criteria":["a"],"owned_files":["a.go"],"complexity":2,"depends_on":[]},
		{"id":"STR-2","title":"B","description":"D","acceptance_criteria":["b"],"owned_files":["b.go"],"complexity":5,"depends_on":["STR-1"]}
	]`

	stories, err := ParseStories(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stories) != 2 {
		t.Fatalf("expected 2 stories, got %d", len(stories))
	}
	if len(stories[1].DependsOn) != 1 || stories[1].DependsOn[0] != "STR-1" {
		t.Errorf("expected STR-2 depends on STR-1, got %v", stories[1].DependsOn)
	}
}

func TestParseValidationValid(t *testing.T) {
	raw := `{"valid":true,"issues":[],"critique":""}`

	result, err := ParseValidation(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid=true")
	}
}

func TestParseValidationWithIssues(t *testing.T) {
	raw := `{"valid":false,"issues":[{"story_id":"STR-1","field":"title","message":"too vague","severity":"error"}],"critique":"improve titles"}`

	result, err := ParseValidation(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Valid {
		t.Error("expected valid=false")
	}
	if len(result.Issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(result.Issues))
	}
	if result.Issues[0].StoryID != "STR-1" {
		t.Errorf("expected issue for STR-1, got %s", result.Issues[0].StoryID)
	}
	if result.Critique != "improve titles" {
		t.Errorf("expected critique 'improve titles', got %q", result.Critique)
	}
}

func TestParseValidationWithCodeFences(t *testing.T) {
	raw := "```json\n" + `{"valid":true,"issues":[],"critique":""}` + "\n```"

	result, err := ParseValidation(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Valid {
		t.Error("expected valid=true")
	}
}

func TestParseValidationInvalidJSON(t *testing.T) {
	_, err := ParseValidation("not valid json")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestStripCodeFencesNoFences(t *testing.T) {
	input := `[{"key":"value"}]`
	result := stripCodeFences(input)
	if result != input {
		t.Errorf("expected no change, got %q", result)
	}
}

func TestStripCodeFencesJSON(t *testing.T) {
	input := "```json\n{\"key\":\"value\"}\n```"
	expected := `{"key":"value"}`
	result := stripCodeFences(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

func TestStripCodeFencesPlain(t *testing.T) {
	input := "```\n{\"key\":\"value\"}\n```"
	expected := `{"key":"value"}`
	result := stripCodeFences(input)
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}
