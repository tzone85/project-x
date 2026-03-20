package planner

import (
	"fmt"

	"github.com/tzone85/project-x/internal/config"
)

// StoryValidator checks story quality against configured criteria.
type StoryValidator struct {
	cfg config.PlanningConfig
}

// NewStoryValidator creates a validator with the given planning config.
func NewStoryValidator(cfg config.PlanningConfig) StoryValidator {
	return StoryValidator{cfg: cfg}
}

// Validate checks all stories against quality criteria and returns issues found.
func (v StoryValidator) Validate(stories []Story) ValidationResult {
	var issues []ValidationIssue

	for _, s := range stories {
		issues = append(issues, v.validateRequiredFields(s)...)
		issues = append(issues, v.validateComplexity(s)...)
	}

	if len(stories) > v.cfg.MaxStoriesPerRequirement {
		issues = append(issues, ValidationIssue{
			Field:    "story_count",
			Message:  fmt.Sprintf("too many stories: %d exceeds max %d", len(stories), v.cfg.MaxStoriesPerRequirement),
			Severity: "error",
		})
	}

	if v.cfg.EnforceFileOwnership {
		issues = append(issues, v.validateFileOwnership(stories)...)
	}

	issues = append(issues, v.validateDependencies(stories)...)

	hasErrors := false
	for _, issue := range issues {
		if issue.Severity == "error" {
			hasErrors = true
			break
		}
	}

	return ValidationResult{
		Valid:  !hasErrors,
		Issues: issues,
	}
}

func (v StoryValidator) validateRequiredFields(s Story) []ValidationIssue {
	var issues []ValidationIssue

	fieldValues := map[string]bool{
		"title":               s.Title != "",
		"description":         s.Description != "",
		"acceptance_criteria": len(s.AcceptanceCriteria) > 0,
		"owned_files":         len(s.OwnedFiles) > 0,
		"complexity":          s.Complexity > 0,
		"depends_on":          true, // empty depends_on is valid (no dependencies)
	}

	for _, field := range v.cfg.RequiredFields {
		if field == "depends_on" {
			continue // depends_on can be empty
		}
		present, known := fieldValues[field]
		if !known || !present {
			issues = append(issues, ValidationIssue{
				StoryID:  s.ID,
				Field:    field,
				Message:  fmt.Sprintf("required field %q is missing or empty", field),
				Severity: "error",
			})
		}
	}

	return issues
}

func (v StoryValidator) validateComplexity(s Story) []ValidationIssue {
	if s.Complexity > v.cfg.MaxStoryComplexity {
		return []ValidationIssue{{
			StoryID:  s.ID,
			Field:    "complexity",
			Message:  fmt.Sprintf("complexity %d exceeds max %d", s.Complexity, v.cfg.MaxStoryComplexity),
			Severity: "error",
		}}
	}
	return nil
}

func (v StoryValidator) validateFileOwnership(stories []Story) []ValidationIssue {
	var issues []ValidationIssue
	fileOwner := make(map[string]string)

	for _, s := range stories {
		for _, f := range s.OwnedFiles {
			if owner, exists := fileOwner[f]; exists {
				issues = append(issues, ValidationIssue{
					StoryID:  s.ID,
					Field:    "owned_files",
					Message:  fmt.Sprintf("file %q already owned by story %q", f, owner),
					Severity: "error",
				})
			} else {
				fileOwner[f] = s.ID
			}
		}
	}

	return issues
}

func (v StoryValidator) validateDependencies(stories []Story) []ValidationIssue {
	var issues []ValidationIssue
	storyIDs := make(map[string]bool)
	for _, s := range stories {
		storyIDs[s.ID] = true
	}

	for _, s := range stories {
		for _, dep := range s.DependsOn {
			if !storyIDs[dep] {
				issues = append(issues, ValidationIssue{
					StoryID:  s.ID,
					Field:    "depends_on",
					Message:  fmt.Sprintf("dependency %q references unknown story", dep),
					Severity: "error",
				})
			}
		}
		// Self-dependency check
		for _, dep := range s.DependsOn {
			if dep == s.ID {
				issues = append(issues, ValidationIssue{
					StoryID:  s.ID,
					Field:    "depends_on",
					Message:  "story depends on itself",
					Severity: "error",
				})
			}
		}
	}

	return issues
}
