package planner

import (
	"fmt"
	"strings"
	"testing"
)

func TestValidate_ValidPlan(t *testing.T) {
	stories := []PlannedStory{
		{ID: "s-1", Title: "Setup DB", Description: "Create tables", AcceptanceCriteria: "Tables exist", Complexity: 3, OwnedFiles: []string{"db.go"}, DependsOn: []string{}},
		{ID: "s-2", Title: "Add API", Description: "REST endpoints", AcceptanceCriteria: "CRUD works", Complexity: 5, OwnedFiles: []string{"api.go"}, DependsOn: []string{"s-1"}},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8, MaxStoriesPerRequirement: 15, EnforceFileOwnership: true})
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestValidate_MissingID(t *testing.T) {
	stories := []PlannedStory{
		{Title: "Do thing", Description: "stuff", AcceptanceCriteria: "ok", Complexity: 3},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8})
	if len(issues) == 0 {
		t.Error("expected issue for missing ID")
	}
}

func TestValidate_MissingTitle(t *testing.T) {
	stories := []PlannedStory{
		{ID: "s-1", Description: "stuff", AcceptanceCriteria: "ok", Complexity: 3},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8})
	if len(issues) == 0 {
		t.Error("expected issue for missing title")
	}
}

func TestValidate_MissingDescription(t *testing.T) {
	stories := []PlannedStory{
		{ID: "s-1", Title: "Do thing", AcceptanceCriteria: "ok", Complexity: 3},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8})
	if len(issues) == 0 {
		t.Error("expected issue for missing description")
	}
}

func TestValidate_MissingAcceptanceCriteria(t *testing.T) {
	stories := []PlannedStory{
		{ID: "s-1", Title: "Do thing", Description: "stuff", Complexity: 3},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8})
	if len(issues) == 0 {
		t.Error("expected issue for missing acceptance criteria")
	}
}

func TestValidate_ComplexityTooHigh(t *testing.T) {
	stories := []PlannedStory{
		{ID: "s-1", Title: "Big task", Description: "huge", AcceptanceCriteria: "ok", Complexity: 13},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8})
	if len(issues) == 0 {
		t.Error("expected issue for high complexity")
	}
}

func TestValidate_ComplexityZero(t *testing.T) {
	stories := []PlannedStory{
		{ID: "s-1", Title: "Task", Description: "desc", AcceptanceCriteria: "ok", Complexity: 0},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8})
	if len(issues) == 0 {
		t.Error("expected issue for zero complexity")
	}
}

func TestValidate_CyclicDependencies(t *testing.T) {
	stories := []PlannedStory{
		{ID: "s-1", Title: "A", Description: "a", AcceptanceCriteria: "ok", Complexity: 3, DependsOn: []string{"s-2"}},
		{ID: "s-2", Title: "B", Description: "b", AcceptanceCriteria: "ok", Complexity: 3, DependsOn: []string{"s-1"}},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8})
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "cycle") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected cycle detection issue, got %v", issues)
	}
}

func TestValidate_OverlappingFileOwnership(t *testing.T) {
	stories := []PlannedStory{
		{ID: "s-1", Title: "A", Description: "a", AcceptanceCriteria: "ok", Complexity: 3, OwnedFiles: []string{"shared.go"}},
		{ID: "s-2", Title: "B", Description: "b", AcceptanceCriteria: "ok", Complexity: 3, OwnedFiles: []string{"shared.go"}},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8, EnforceFileOwnership: true})
	if len(issues) == 0 {
		t.Error("expected issue for overlapping file ownership")
	}
}

func TestValidate_OverlappingFileOwnership_NotEnforced(t *testing.T) {
	stories := []PlannedStory{
		{ID: "s-1", Title: "A", Description: "a", AcceptanceCriteria: "ok", Complexity: 3, OwnedFiles: []string{"shared.go"}},
		{ID: "s-2", Title: "B", Description: "b", AcceptanceCriteria: "ok", Complexity: 3, OwnedFiles: []string{"shared.go"}},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8, EnforceFileOwnership: false})
	if len(issues) != 0 {
		t.Errorf("expected no issues when file ownership not enforced, got %v", issues)
	}
}

func TestValidate_TooManyStories(t *testing.T) {
	var stories []PlannedStory
	for i := 0; i < 20; i++ {
		stories = append(stories, PlannedStory{
			ID:                 fmt.Sprintf("s-%d", i),
			Title:              fmt.Sprintf("Story %d", i),
			Description:        "desc",
			AcceptanceCriteria: "ac",
			Complexity:         2,
		})
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8, MaxStoriesPerRequirement: 15})
	if len(issues) == 0 {
		t.Error("expected issue for too many stories")
	}
}

func TestValidate_DuplicateIDs(t *testing.T) {
	stories := []PlannedStory{
		{ID: "s-1", Title: "A", Description: "a", AcceptanceCriteria: "ok", Complexity: 3},
		{ID: "s-1", Title: "B", Description: "b", AcceptanceCriteria: "ok", Complexity: 3},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8})
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "duplicate") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected duplicate ID issue, got %v", issues)
	}
}

func TestValidate_UnknownDependency(t *testing.T) {
	stories := []PlannedStory{
		{ID: "s-1", Title: "A", Description: "a", AcceptanceCriteria: "ok", Complexity: 3, DependsOn: []string{"s-999"}},
	}
	issues := Validate(stories, PlannerConfig{MaxStoryComplexity: 8})
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "unknown") || strings.Contains(issue, "s-999") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected unknown dependency issue, got %v", issues)
	}
}

func TestValidate_EmptyStories(t *testing.T) {
	issues := Validate(nil, PlannerConfig{MaxStoryComplexity: 8})
	if len(issues) == 0 {
		t.Error("expected issue for empty stories")
	}
}
