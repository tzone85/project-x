package planner

import (
	"testing"

	"github.com/tzone85/project-x/internal/config"
)

func defaultPlanningConfig() config.PlanningConfig {
	return config.PlanningConfig{
		RequiredFields: []string{
			"title", "description", "acceptance_criteria",
			"owned_files", "complexity", "depends_on",
		},
		MaxStoryComplexity:       8,
		MaxStoriesPerRequirement: 15,
		EnforceFileOwnership:     true,
	}
}

func validStory(id string) Story {
	return Story{
		ID:                 id,
		Title:              "Implement feature " + id,
		Description:        "Description for " + id,
		AcceptanceCriteria: []string{"It works"},
		OwnedFiles:         []string{id + ".go"},
		Complexity:         3,
		DependsOn:          []string{},
	}
}

func TestValidatorValidStories(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	stories := []Story{validStory("STR-1"), validStory("STR-2")}

	result := v.Validate(stories)

	if !result.Valid {
		t.Errorf("expected valid result, got issues: %v", result.Issues)
	}
	if len(result.Issues) != 0 {
		t.Errorf("expected no issues, got %d", len(result.Issues))
	}
}

func TestValidatorMissingTitle(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s := validStory("STR-1")
	s.Title = ""

	result := v.Validate([]Story{s})

	if result.Valid {
		t.Error("expected invalid result for missing title")
	}
	assertHasIssue(t, result.Issues, "STR-1", "title")
}

func TestValidatorMissingDescription(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s := validStory("STR-1")
	s.Description = ""

	result := v.Validate([]Story{s})

	if result.Valid {
		t.Error("expected invalid result for missing description")
	}
	assertHasIssue(t, result.Issues, "STR-1", "description")
}

func TestValidatorMissingAcceptanceCriteria(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s := validStory("STR-1")
	s.AcceptanceCriteria = nil

	result := v.Validate([]Story{s})

	if result.Valid {
		t.Error("expected invalid result for missing acceptance_criteria")
	}
	assertHasIssue(t, result.Issues, "STR-1", "acceptance_criteria")
}

func TestValidatorMissingOwnedFiles(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s := validStory("STR-1")
	s.OwnedFiles = nil

	result := v.Validate([]Story{s})

	if result.Valid {
		t.Error("expected invalid result for missing owned_files")
	}
	assertHasIssue(t, result.Issues, "STR-1", "owned_files")
}

func TestValidatorZeroComplexity(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s := validStory("STR-1")
	s.Complexity = 0

	result := v.Validate([]Story{s})

	if result.Valid {
		t.Error("expected invalid result for zero complexity")
	}
	assertHasIssue(t, result.Issues, "STR-1", "complexity")
}

func TestValidatorComplexityExceedsMax(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s := validStory("STR-1")
	s.Complexity = 13

	result := v.Validate([]Story{s})

	if result.Valid {
		t.Error("expected invalid result for complexity exceeding max")
	}
	assertHasIssue(t, result.Issues, "STR-1", "complexity")
}

func TestValidatorComplexityAtMax(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s := validStory("STR-1")
	s.Complexity = 8

	result := v.Validate([]Story{s})

	if !result.Valid {
		t.Errorf("expected valid result at max complexity, got issues: %v", result.Issues)
	}
}

func TestValidatorTooManyStories(t *testing.T) {
	cfg := defaultPlanningConfig()
	cfg.MaxStoriesPerRequirement = 2
	v := NewStoryValidator(cfg)

	stories := []Story{validStory("STR-1"), validStory("STR-2"), validStory("STR-3")}

	result := v.Validate(stories)

	if result.Valid {
		t.Error("expected invalid result for too many stories")
	}
	assertHasIssue(t, result.Issues, "", "story_count")
}

func TestValidatorFileOwnershipConflict(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s1 := validStory("STR-1")
	s2 := validStory("STR-2")
	s2.OwnedFiles = s1.OwnedFiles // same file

	result := v.Validate([]Story{s1, s2})

	if result.Valid {
		t.Error("expected invalid result for file ownership conflict")
	}
	assertHasIssue(t, result.Issues, "STR-2", "owned_files")
}

func TestValidatorFileOwnershipDisabled(t *testing.T) {
	cfg := defaultPlanningConfig()
	cfg.EnforceFileOwnership = false
	v := NewStoryValidator(cfg)

	s1 := validStory("STR-1")
	s2 := validStory("STR-2")
	s2.OwnedFiles = s1.OwnedFiles // same file, but enforcement disabled

	result := v.Validate([]Story{s1, s2})

	if !result.Valid {
		t.Errorf("expected valid when ownership enforcement disabled, got issues: %v", result.Issues)
	}
}

func TestValidatorUnknownDependency(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s := validStory("STR-1")
	s.DependsOn = []string{"STR-999"}

	result := v.Validate([]Story{s})

	if result.Valid {
		t.Error("expected invalid result for unknown dependency")
	}
	assertHasIssue(t, result.Issues, "STR-1", "depends_on")
}

func TestValidatorSelfDependency(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s := validStory("STR-1")
	s.DependsOn = []string{"STR-1"}

	result := v.Validate([]Story{s})

	if result.Valid {
		t.Error("expected invalid result for self dependency")
	}
	assertHasIssue(t, result.Issues, "STR-1", "depends_on")
}

func TestValidatorValidDependency(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s1 := validStory("STR-1")
	s2 := validStory("STR-2")
	s2.DependsOn = []string{"STR-1"}

	result := v.Validate([]Story{s1, s2})

	if !result.Valid {
		t.Errorf("expected valid for correct dependency, got issues: %v", result.Issues)
	}
}

func TestValidatorEmptyDependsOnIsValid(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s := validStory("STR-1")
	s.DependsOn = nil

	result := v.Validate([]Story{s})

	if !result.Valid {
		t.Errorf("expected valid for nil depends_on, got issues: %v", result.Issues)
	}
}

func TestValidatorMultipleIssuesAccumulated(t *testing.T) {
	v := NewStoryValidator(defaultPlanningConfig())
	s := Story{ID: "STR-1"} // missing everything

	result := v.Validate([]Story{s})

	if result.Valid {
		t.Error("expected invalid for story missing all fields")
	}
	if len(result.Issues) < 4 {
		t.Errorf("expected at least 4 issues, got %d: %v", len(result.Issues), result.Issues)
	}
}

func assertHasIssue(t *testing.T, issues []ValidationIssue, storyID, field string) {
	t.Helper()
	for _, issue := range issues {
		if (storyID == "" || issue.StoryID == storyID) && issue.Field == field {
			return
		}
	}
	t.Errorf("expected issue for story=%q field=%q, not found in %v", storyID, field, issues)
}
