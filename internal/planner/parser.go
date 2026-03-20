package planner

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ParseStories extracts a []Story from an LLM response string.
// It handles responses that may contain markdown code fences.
func ParseStories(raw string) ([]Story, error) {
	cleaned := stripCodeFences(raw)
	cleaned = strings.TrimSpace(cleaned)

	var stories []Story
	if err := json.Unmarshal([]byte(cleaned), &stories); err != nil {
		return nil, fmt.Errorf("parsing stories JSON: %w", err)
	}

	if len(stories) == 0 {
		return nil, fmt.Errorf("decomposition produced zero stories")
	}

	return stories, nil
}

// ParseValidation extracts a ValidationResult from an LLM response string.
func ParseValidation(raw string) (ValidationResult, error) {
	cleaned := stripCodeFences(raw)
	cleaned = strings.TrimSpace(cleaned)

	var result ValidationResult
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return ValidationResult{}, fmt.Errorf("parsing validation JSON: %w", err)
	}

	return result, nil
}

// stripCodeFences removes markdown code fences from LLM output.
func stripCodeFences(s string) string {
	s = strings.TrimSpace(s)

	// Remove ```json ... ``` or ``` ... ```
	if strings.HasPrefix(s, "```") {
		// Find end of first line (the opening fence)
		idx := strings.Index(s, "\n")
		if idx >= 0 {
			s = s[idx+1:]
		}
		// Remove closing fence
		if lastIdx := strings.LastIndex(s, "```"); lastIdx >= 0 {
			s = s[:lastIdx]
		}
	}

	return strings.TrimSpace(s)
}
